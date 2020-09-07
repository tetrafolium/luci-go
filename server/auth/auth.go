// Copyright 2015 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"net"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/trace"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/auth/delegation"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/router"
)

var (
	// Authenticate errors (must be grpc-tagged).

	// ErrNotConfigured is returned by Authenticate and other functions if the
	// context wasn't previously initialized via 'Initialize'.
	ErrNotConfigured = errors.New("auth: the library is not properly configured", grpcutil.InternalTag)

	// ErrBadClientID is returned by Authenticate if caller is using
	// non-whitelisted OAuth2 client. More info is in the log.
	ErrBadClientID = errors.New("auth: OAuth client_id is not whitelisted", grpcutil.PermissionDeniedTag)

	// ErrBadAudience is returned by Authenticate if token's audience is unknown.
	ErrBadAudience = errors.New("auth: bad token audience", grpcutil.PermissionDeniedTag)

	// ErrBadRemoteAddr is returned by Authenticate if request's remote_addr can't
	// be parsed.
	ErrBadRemoteAddr = errors.New("auth: bad remote addr", grpcutil.InternalTag)

	// ErrIPNotWhitelisted is returned when an account is restricted by an IP
	// whitelist and request's remote_addr is not in it.
	ErrIPNotWhitelisted = errors.New("auth: IP is not whitelisted", grpcutil.PermissionDeniedTag)

	// ErrProjectHeaderForbidden is returned by Authenticate if an unknown caller
	// tries to use X-Luci-Project header. Only a whitelisted set of callers
	// are allowed to use this header, see InternalServicesGroup.
	ErrProjectHeaderForbidden = errors.New("auth: the caller is not allowed to use X-Luci-Project", grpcutil.PermissionDeniedTag)

	// Other errors.

	// ErrNoUsersAPI is returned by LoginURL and LogoutURL if none of
	// the authentication methods support UsersAPI.
	ErrNoUsersAPI = errors.New("auth: methods do not support login or logout URL")

	// ErrNoForwardableCreds is returned by GetRPCTransport when attempting to
	// forward credentials (via AsCredentialsForwarder) that are not forwardable.
	ErrNoForwardableCreds = errors.New("auth: no forwardable credentials in the context")
)

const (
	// InternalServicesGroup is a name of a group with service accounts of LUCI
	// microservices of the current LUCI deployment (and only them!).
	//
	// Accounts in this group are allowed to use X-Luci-Project header to specify
	// that RPCs are done in a context of some particular project. For such
	// requests CurrentIdentity() == 'project:<X-Luci-Project value>'.
	//
	// This group should contain only **fully trusted** services, deployed and
	// managed by the LUCI deployment administrators. Adding "random" services
	// here is a security risk, since they will be able to impersonate any LUCI
	// project.
	InternalServicesGroup = "auth-luci-services"
)

// Method implements a particular kind of low-level authentication mechanism.
//
// It may also optionally implement a bunch of other interfaces:
//   UsersAPI: if the method supports login and logout URLs.
//   Warmable: if the method supports warm up.
//   HasHandlers: if the method needs to install HTTP handlers.
//
// Methods are not usually used directly, but passed to Authenticator{...} that
// knows how to apply them.
type Method interface {
	// Authenticate extracts user information from the incoming request.
	//
	// It returns:
	//   * (*User, nil) on success.
	//   * (nil, nil) if the method is not applicable.
	//   * (nil, error) if the method is applicable, but credentials are invalid.
	//
	// The returned error may be tagged with an grpcutil error tag. Its code will
	// be used to derive the response status code. Internal error messages (e.g.
	// ones tagged with grpcutil.InternalTag or similar) are logged, but not sent
	// to clients. All other errors are sent to clients as is.
	Authenticate(context.Context, *http.Request) (*User, error)
}

// UsersAPI may be additionally implemented by Method if it supports login and
// logout URLs.
type UsersAPI interface {
	// LoginURL returns a URL that, when visited, prompts the user to sign in,
	// then redirects the user to the URL specified by dest.
	LoginURL(ctx context.Context, dest string) (string, error)

	// LogoutURL returns a URL that, when visited, signs the user out,
	// then redirects the user to the URL specified by dest.
	LogoutURL(ctx context.Context, dest string) (string, error)
}

// Warmable may be additionally implemented by Method if it supports warm up.
type Warmable interface {
	// Warmup may be called to precache the data needed by the method.
	//
	// There's no guarantee when it will be called or if it will be called at all.
	// Should always do best-effort initialization. Errors are logged and ignored.
	Warmup(ctx context.Context) error
}

// HasHandlers may be additionally implemented by Method if it needs to
// install HTTP handlers.
type HasHandlers interface {
	// InstallHandlers installs necessary HTTP handlers into the router.
	InstallHandlers(r *router.Router, base router.MiddlewareChain)
}

// UserCredentialsGetter may be additionally implemented by Method if it knows
// how to extract end-user credentials from the incoming request. Currently
// understands only OAuth2 tokens.
type UserCredentialsGetter interface {
	// GetUserCredentials extracts an OAuth access token from the incoming request
	// or returns an error if it isn't possible.
	//
	// May omit token's expiration time if it isn't known.
	//
	// Guaranteed to be called only after the successful authentication, so it
	// doesn't have to recheck the validity of the token.
	GetUserCredentials(context.Context, *http.Request) (*oauth2.Token, error)
}

// User represents identity and profile of a user.
type User struct {
	// Identity is identity string of the user (may be AnonymousIdentity).
	// If User is returned by Authenticate(...), Identity string is always present
	// and valid.
	Identity identity.Identity `json:"identity,omitempty"`

	// Superuser is true if the user is site-level administrator. For example, on
	// GAE this bit is set for GAE-level administrators. Optional, default false.
	Superuser bool `json:"superuser,omitempty"`

	// Email is email of the user. Optional, default "". Don't use it as a key
	// in various structures. Prefer to use Identity() instead (it is always
	// available).
	Email string `json:"email,omitempty"`

	// Name is full name of the user. Optional, default "".
	Name string `json:"name,omitempty"`

	// Picture is URL of the user avatar. Optional, default "".
	Picture string `json:"picture,omitempty"`

	// ClientID is the ID of the pre-registered OAuth2 client so its identity can
	// be verified. Used only by authentication methods based on OAuth2.
	// See https://developers.google.com/console/help/#generatingoauth2 for more.
	ClientID string `json:"client_id,omitempty"`
}

// Authenticator performs authentication of incoming requests.
//
// It is a stateless object configured with a list of methods to try when
// authenticating incoming requests. It implements Authenticate method that
// performs high-level authentication logic using the provided list of low-level
// auth methods.
//
// Note that most likely you don't need to instantiate this object directly.
// Use Authenticate middleware instead. Authenticator is exposed publicly only
// to be used in advanced cases, when you need to fine-tune authentication
// behavior.
type Authenticator struct {
	Methods []Method // a list of authentication methods to try
}

// GetMiddleware returns a middleware that uses this Authenticator for
// authentication.
//
// It uses a.Authenticate internally and handles errors appropriately.
func (a *Authenticator) GetMiddleware() router.Middleware {
	return func(c *router.Context, next router.Handler) {
		ctx, err := a.Authenticate(c.Context, c.Request)
		if err != nil {
			code, ok := grpcutil.Tag.In(err)
			if !ok {
				if transient.Tag.In(err) {
					code = codes.Internal
				} else {
					code = codes.Unauthenticated
				}
			}
			replyError(c.Context, c.Writer, grpcutil.CodeStatus(code), err)
		} else {
			c.Context = ctx
			next(c)
		}
	}
}

// Authenticate authenticates the requests and adds State into the context.
//
// Returns an error if credentials are provided, but invalid. If no credentials
// are provided (i.e. the request is anonymous), finishes successfully, but in
// that case CurrentIdentity() returns AnonymousIdentity.
//
// The returned error may be tagged with an grpcutil error tag. Its code should
// be used to derive the response status code. Internal error messages (e.g.
// ones tagged with grpcutil.InternalTag or similar) should be logged, but not
// sent to clients. All other errors should be sent to clients as is.
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request) (_ context.Context, err error) {
	tracedCtx, span := trace.StartSpan(ctx, "github.com/tetrafolium/luci-go/server/auth.Authenticate")
	report := durationReporter(tracedCtx, authenticateDuration)

	// This variable is changed throughout the function's execution. It it used
	// in the defer to figure out at what stage the call failed.
	stage := ""

	// This defer reports the outcome of the authentication to the monitoring.
	defer func() {
		switch {
		case err == nil:
			report(nil, "SUCCESS")
		case err == ErrNotConfigured:
			report(err, "ERROR_NOT_CONFIGURED")
		case err == ErrBadClientID:
			report(err, "ERROR_FORBIDDEN_OAUTH_CLIENT")
		case err == ErrBadAudience:
			report(err, "ERROR_FORBIDDEN_AUDIENCE")
		case err == ErrBadRemoteAddr:
			report(err, "ERROR_BAD_REMOTE_ADDR")
		case err == ErrIPNotWhitelisted:
			report(err, "ERROR_FORBIDDEN_IP")
		case err == ErrProjectHeaderForbidden:
			report(err, "ERROR_PROJECT_HEADER_FORBIDDEN")
		case transient.Tag.In(err):
			report(err, "ERROR_TRANSIENT_IN_"+stage)
		default:
			report(err, "ERROR_IN_"+stage)
		}
		span.End(err)
	}()

	// We will need working DB factory below to check IP whitelist.
	cfg := getConfig(tracedCtx)
	if cfg == nil || cfg.DBProvider == nil || len(a.Methods) == 0 {
		return nil, ErrNotConfigured
	}

	// The future state that will be placed into the context.
	s := state{authenticator: a, endUserErr: ErrNoForwardableCreds}

	// Pick the first authentication method that applies.
	stage = "AUTH"
	for _, m := range a.Methods {
		var err error
		if s.user, err = m.Authenticate(tracedCtx, r); err != nil {
			return nil, err
		}
		if s.user != nil {
			if err = s.user.Identity.Validate(); err != nil {
				stage = "ID_REGEXP_CHECK"
				return nil, err
			}
			s.method = m
			break
		}
	}

	// If no authentication method is applicable, default to anonymous identity.
	if s.method == nil {
		s.user = &User{Identity: identity.AnonymousIdentity}
	}

	// peerIdent always matches the identity of a remote peer. It may end up being
	// different from s.user.Identity if the delegation tokens or project
	// identities are used (see below). They affect s.user.Identity but don't
	// touch s.peerIdent.
	s.peerIdent = s.user.Identity

	// Grab a snapshot of auth DB to use it consistently for the duration of this
	// request.
	stage = "AUTHDB_FETCH"
	s.db, err = cfg.DBProvider(tracedCtx)
	if err != nil {
		return nil, err
	}

	// If using OAuth2, make sure the ClientID is whitelisted.
	if s.user.ClientID != "" {
		stage = "OAUTH_WHITELIST"
		if err := checkClientIDWhitelist(tracedCtx, cfg, s.db, s.user.Email, s.user.ClientID); err != nil {
			return nil, err
		}
	}

	// Extract peer's IP address and, if necessary, check it against a whitelist.
	stage = "IP_WHITELIST"
	if s.peerIP, err = checkPeerIP(tracedCtx, cfg, s.db, r, s.peerIdent); err != nil {
		return nil, err
	}

	// Check X-Delegation-Token-V1 and X-Luci-Project headers. They are used in
	// LUCI-specific protocols to allow LUCI micro-services to act on behalf of
	// end-users or projects.
	if token := r.Header.Get(delegation.HTTPHeaderName); token != "" {
		stage = "DELEGATION_TOKEN_CHECK"
		if s.user, err = checkDelegationToken(tracedCtx, cfg, s.db, token, s.peerIdent); err != nil {
			return nil, err
		}
	} else if project := r.Header.Get(XLUCIProjectHeader); project != "" {
		stage = "PROJECT_HEADER_CHECK"
		if s.user, err = checkProjectHeader(tracedCtx, s.db, project, s.peerIdent); err != nil {
			return nil, err
		}
	} else {
		// If not using LUCI-specific protocols, grab the end user creds in case we
		// want to forward them later in GetRPCTransport(AsCredentialsForwarder).
		if credsGetter, _ := s.method.(UserCredentialsGetter); credsGetter != nil {
			s.endUserTok, s.endUserErr = credsGetter.GetUserCredentials(tracedCtx, r)
		}
	}

	// Inject the auth state into the original context (not the traced one).
	return WithState(ctx, &s), nil
}

// usersAPI returns implementation of UsersAPI by examining Methods.
//
// Returns nil if none of Methods implement UsersAPI.
func (a *Authenticator) usersAPI() UsersAPI {
	for _, m := range a.Methods {
		if api, ok := m.(UsersAPI); ok {
			return api
		}
	}
	return nil
}

// LoginURL returns a URL that, when visited, prompts the user to sign in,
// then redirects the user to the URL specified by dest.
//
// Returns ErrNoUsersAPI if none of the authentication methods support login
// URLs.
func (a *Authenticator) LoginURL(ctx context.Context, dest string) (string, error) {
	if api := a.usersAPI(); api != nil {
		return api.LoginURL(ctx, dest)
	}
	return "", ErrNoUsersAPI
}

// LogoutURL returns a URL that, when visited, signs the user out, then
// redirects the user to the URL specified by dest.
//
// Returns ErrNoUsersAPI if none of the authentication methods support login
// URLs.
func (a *Authenticator) LogoutURL(ctx context.Context, dest string) (string, error) {
	if api := a.usersAPI(); api != nil {
		return api.LogoutURL(ctx, dest)
	}
	return "", ErrNoUsersAPI
}

////

// replyError logs the error and writes a response to ResponseWriter.
//
// For codes < 500, the error is logged at Warning level and written to the
// response as is. For codes >= 500 the error is logged at Error level and
// the generic error message is written instead.
func replyError(ctx context.Context, rw http.ResponseWriter, code int, err error) {
	if code < 500 {
		logging.Warningf(ctx, "HTTP %d: %s", code, err)
		http.Error(rw, err.Error(), code)
	} else {
		logging.Errorf(ctx, "HTTP %d: %s", code, err)
		http.Error(rw, http.StatusText(code), code)
	}
}

// checkClientIDWhitelist returns nil if the clientID is allowed, ErrBadClientID
// if not, and a transient errors if the check itself failed.
func checkClientIDWhitelist(ctx context.Context, cfg *Config, db authdb.DB, email, clientID string) error {
	// Check the global whitelist in the AuthDB.
	switch valid, err := db.IsAllowedOAuthClientID(ctx, email, clientID); {
	case err != nil:
		return errors.Annotate(err, "failed to check client ID whitelist").Tag(transient.Tag).Err()
	case valid:
		return nil
	}

	// It may be an app-specific client ID supplied via cfg.FrontendClientID.
	if cfg.FrontendClientID != nil {
		switch frontendClientID, err := cfg.FrontendClientID(ctx); {
		case err != nil:
			return errors.Annotate(err, "failed to grab frontend client ID").Tag(transient.Tag).Err()
		case clientID == frontendClientID:
			return nil
		}
	}

	logging.Errorf(ctx, "auth: %q is using client_id %q not in the whitelist", email, clientID)
	return ErrBadClientID
}

// checkPeerIP parses the caller IP address and checks it against a whitelist
// (if necessary). Returns ErrBadRemoteAddr if the IP is malformed,
// ErrIPNotWhitelisted if the IP is not whitelisted or a transient error if the
// check itself failed.
func checkPeerIP(ctx context.Context, cfg *Config, db authdb.DB, r *http.Request, peerID identity.Identity) (net.IP, error) {
	remoteAddr := r.RemoteAddr
	if cfg.EndUserIP != nil {
		remoteAddr = cfg.EndUserIP(r)
	}
	peerIP, err := parseRemoteIP(remoteAddr)
	if err != nil {
		logging.Errorf(ctx, "auth: bad remote_addr %q in a call from %q - %s", remoteAddr, peerID, err)
		return nil, ErrBadRemoteAddr
	}

	// Some callers may be constrained by an IP whitelist.
	switch ipWhitelist, err := db.GetWhitelistForIdentity(ctx, peerID); {
	case err != nil:
		return nil, errors.Annotate(err, "failed to get IP whitelist for identity %q", peerID).Tag(transient.Tag).Err()
	case ipWhitelist != "":
		switch whitelisted, err := db.IsInWhitelist(ctx, peerIP, ipWhitelist); {
		case err != nil:
			return nil, errors.Annotate(err, "failed to check IP %s is in the whitelist %q", peerIP, ipWhitelist).Tag(transient.Tag).Err()
		case !whitelisted:
			return nil, ErrIPNotWhitelisted
		}
	}

	return peerIP, nil
}

// checkDelegationToken checks correctness of a delegation token and returns
// a delegated *User.
func checkDelegationToken(ctx context.Context, cfg *Config, db authdb.DB, token string, peerID identity.Identity) (*User, error) {
	// Log the token fingerprint (even before parsing the token), it can be used
	// to grab the info about the token from the token server logs.
	logging.Fields{
		"fingerprint": tokenFingerprint(token),
	}.Debugf(ctx, "auth: Received delegation token")

	// Need to grab our own identity to verify that the delegation token is
	// minted for consumption by us and not some other service.
	ownServiceIdentity, err := getOwnServiceIdentity(ctx, cfg.Signer)
	if err != nil {
		return nil, err
	}
	delegatedIdentity, err := delegation.CheckToken(ctx, delegation.CheckTokenParams{
		Token:                token,
		PeerID:               peerID,
		CertificatesProvider: db,
		GroupsChecker:        db,
		OwnServiceIdentity:   ownServiceIdentity,
	})
	if err != nil {
		return nil, err
	}

	// Log that peerID is pretending to be delegatedIdentity.
	logging.Fields{
		"peerID":      peerID,
		"delegatedID": delegatedIdentity,
	}.Debugf(ctx, "auth: Using delegation")

	return &User{Identity: delegatedIdentity}, nil
}

// checkProjectHeader verifies the caller is allowed to use X-Luci-Project
// mechanism and returns a *User (with project-scoped identity) to use for
// the request.
func checkProjectHeader(ctx context.Context, db authdb.DB, project string, peerID identity.Identity) (*User, error) {
	// See comment for InternalServicesGroup.
	switch yes, err := db.IsMember(ctx, peerID, []string{InternalServicesGroup}); {
	case err != nil:
		return nil, errors.Annotate(err, "error when checking if %q is in %q", peerID, InternalServicesGroup).Tag(transient.Tag).Err()
	case !yes:
		return nil, ErrProjectHeaderForbidden
	}

	// Verify the actual value passes the regexp check.
	projIdent, err := identity.MakeIdentity("project:" + project)
	if err != nil {
		return nil, errors.Annotate(err, "bad %s", XLUCIProjectHeader).Err()
	}

	// Log that peerID is using project-scoped identity.
	logging.Fields{
		"peerID":    peerID,
		"projectID": projIdent,
	}.Debugf(ctx, "auth: Using project identity")

	return &User{Identity: projIdent}, nil
}

// getOwnServiceIdentity returns 'service:<appID>' identity of the current
// service.
func getOwnServiceIdentity(ctx context.Context, signer signing.Signer) (identity.Identity, error) {
	if signer == nil {
		return "", ErrNotConfigured
	}
	switch serviceInfo, err := signer.ServiceInfo(ctx); {
	case err != nil:
		return "", err
	case serviceInfo.AppID == "":
		return "", errors.Reason("auth: don't known our own app ID to check the delegation token is for us").Err()
	default:
		return identity.MakeIdentity("service:" + serviceInfo.AppID)
	}
}
