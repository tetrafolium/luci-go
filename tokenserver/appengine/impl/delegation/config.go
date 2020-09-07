// Copyright 2016 The LUCI Authors.
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

package delegation

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/config/validation"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/identityset"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/policy"
)

// delegationCfg is name of the main config file with the policy.
//
// Also used as a name for the imported configs in the datastore, so change it
// very carefully.
const delegationCfg = "delegation.cfg"

const (
	// Requestor is magical token that may be used in the config and requests as
	// a substitute for caller's ID.
	//
	// See config.proto for more info.
	Requestor = "REQUESTOR"

	// Projects is a magical token that can be used in allowed_to_impersonate to
	// indicate that the caller can impersonate "project:*" identities.
	//
	// TODO(vadimsh): Get rid of it.
	Projects = "PROJECTS"
)

// Rules is queryable representation of delegation.cfg rules.
type Rules struct {
	revision   string            // config revision this policy is imported from
	rules      []*delegationRule // preprocessed policy rules
	requestors *identityset.Set  // union of all 'Requestor' fields in all rules
}

// RulesQuery contains parameters to match against the delegation rules.
//
// Used by 'FindMatchingRule'.
type RulesQuery struct {
	Requestor identity.Identity // who is requesting the token
	Delegator identity.Identity // what identity will be delegated/impersonated
	Audience  *identityset.Set  // the requested audience set (delegatees)
	Services  *identityset.Set  // the requested target services set
}

// delegationRule is preprocessed admin.DelegationRule message.
//
// This object is used by 'FindMatchingRule'.
type delegationRule struct {
	rule *admin.DelegationRule // the original unaltered rule proto

	requestors *identityset.Set // matched to RulesQuery.Requestor
	delegators *identityset.Set // matched to RulesQuery.Delegator
	audience   *identityset.Set // matched to RulesQuery.Audience
	services   *identityset.Set // matched to RulesQuery.Services

	addRequestorAsDelegator bool // if true, add RulesQuery.Requestor to 'delegators' set
	addRequestorToAudience  bool // if true, add RulesQuery.Requestor to 'audience' set
	addProjectsAsDelegators bool // if true, add 'project:*' to 'delegators' set
}

// RulesCache is a stateful object with parsed delegation.cfg rules.
//
// It uses policy.Policy internally to manage datastore-cached copy of imported
// delegation configs.
//
// Use NewRulesCache() to create a new instance. Each instance owns its own
// in-memory cache, but uses same shared datastore cache.
//
// There's also a process global instance of RulesCache (GlobalRulesCache var)
// which is used by the main process. Unit tests don't use it though to avoid
// relying on shared state.
type RulesCache struct {
	policy policy.Policy // holds cached *parsedRules
}

// GlobalRulesCache is the process-wide rules cache.
var GlobalRulesCache = NewRulesCache()

// NewRulesCache properly initializes RulesCache instance.
func NewRulesCache() *RulesCache {
	return &RulesCache{
		policy: policy.Policy{
			Name:     delegationCfg,        // used as part of datastore keys
			Fetch:    fetchConfigs,         // see below
			Validate: validateConfigBundle, // see config_validation.go
			Prepare:  prepareRules,         // see below
		},
	}
}

// ImportConfigs refetches delegation.cfg and updates datastore copy of it.
//
// Called from cron.
func (rc *RulesCache) ImportConfigs(c context.Context) (rev string, err error) {
	return rc.policy.ImportConfigs(c)
}

// SetupConfigValidation registers the config validation rules.
func (rc *RulesCache) SetupConfigValidation(rules *validation.RuleSet) {
	rules.Add("services/${appid}", delegationCfg, func(ctx *validation.Context, configSet, path string, content []byte) error {
		cfg := &admin.DelegationPermissions{}
		if err := proto.UnmarshalText(string(content), cfg); err != nil {
			ctx.Errorf("not a valid DelegationPermissions proto message - %s", err)
		} else {
			validateDelegationCfg(ctx, cfg)
		}
		return nil
	})
}

// Rules returns in-memory copy of delegation rules, ready for querying.
func (rc *RulesCache) Rules(c context.Context) (*Rules, error) {
	q, err := rc.policy.Queryable(c)
	if err != nil {
		return nil, err
	}
	return q.(*Rules), nil
}

// fetchConfigs loads proto messages with rules from the config.
func fetchConfigs(c context.Context, f policy.ConfigFetcher) (policy.ConfigBundle, error) {
	cfg := &admin.DelegationPermissions{}
	if err := f.FetchTextProto(c, delegationCfg, cfg); err != nil {
		return nil, err
	}
	return policy.ConfigBundle{delegationCfg: cfg}, nil
}

// prepareRules converts validated configs into *Rules.
//
// Returns them as policy.Queryable object to satisfy policy.Policy API.
func prepareRules(c context.Context, cfg policy.ConfigBundle, revision string) (policy.Queryable, error) {
	parsed, ok := cfg[delegationCfg].(*admin.DelegationPermissions)
	if !ok {
		return nil, fmt.Errorf("wrong type of delegation.cfg - %T", cfg[delegationCfg])
	}

	rules := make([]*delegationRule, 0, len(parsed.Rules)+1)
	for _, msg := range parsed.Rules {
		rule, err := makeDelegationRule(c, msg)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	// Add an implicit rule that allows trusted LUCI microservices to grab
	// delegation tokens for 'project:*' identities. "auth-luci-services" is a
	// magical group also mentioned in luci-py's components.auth.
	//
	// TODO(vadimsh): Currently relied on by Buildbucket. Buildbucket should
	// switch to using 'project:...' identities directly without going through
	// the token server.
	rule, err := makeDelegationRule(c, &admin.DelegationRule{
		Name:                 "allow-project-identities",
		Requestor:            []string{"group:auth-luci-services"},
		AllowedToImpersonate: []string{Projects},
		AllowedAudience:      []string{Requestor},
		TargetService:        []string{"*"},
		MaxValidityDuration:  86400,
	})
	if err != nil {
		panic(err) // should be impossible, this is a hardcoded rule
	}
	rules = append(rules, rule)

	requestors := make([]*identityset.Set, len(rules))
	for i, r := range rules {
		requestors[i] = r.requestors
	}

	return &Rules{
		revision:   revision,
		rules:      rules,
		requestors: identityset.Union(requestors...),
	}, nil
}

// makeDelegationRule preprocesses admin.DelegationRule proto.
//
// It also double checks that the rule is passing validation. The check may
// fail if new code uses old configs, still stored in the datastore.
func makeDelegationRule(c context.Context, rule *admin.DelegationRule) (*delegationRule, error) {
	ctx := &validation.Context{Context: c}
	validateRule(ctx, rule.Name, rule)
	if err := ctx.Finalize(); err != nil {
		return nil, err
	}

	// The main validation step has been done above. Here we just assert that
	// everything looks sane (it should). See corresponding chunks of
	// 'ValidateRule' code.
	requestors, err := identityset.FromStrings(rule.Requestor, nil)
	if err != nil {
		panic(err)
	}
	delegators, err := identityset.FromStrings(rule.AllowedToImpersonate, skipRequestorOrProjects)
	if err != nil {
		panic(err)
	}
	audience, err := identityset.FromStrings(rule.AllowedAudience, skipRequestor)
	if err != nil {
		panic(err)
	}
	services, err := identityset.FromStrings(rule.TargetService, nil)
	if err != nil {
		panic(err)
	}

	return &delegationRule{
		rule:                    rule,
		requestors:              requestors,
		delegators:              delegators,
		audience:                audience,
		services:                services,
		addRequestorAsDelegator: sliceHasString(rule.AllowedToImpersonate, Requestor),
		addRequestorToAudience:  sliceHasString(rule.AllowedAudience, Requestor),
		addProjectsAsDelegators: sliceHasString(rule.AllowedToImpersonate, Projects),
	}, nil
}

func skipRequestor(s string) bool {
	return s == Requestor
}

func skipRequestorOrProjects(s string) bool {
	return s == Requestor || s == Projects
}

func sliceHasString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ConfigRevision is part of policy.Queryable interface.
func (r *Rules) ConfigRevision() string {
	return r.revision
}

// IsAuthorizedRequestor returns true if the caller belongs to 'requestor' set
// of at least one rule.
func (r *Rules) IsAuthorizedRequestor(c context.Context, id identity.Identity) (bool, error) {
	return r.requestors.IsMember(c, id)
}

// FindMatchingRule finds one and only one rule matching the query.
//
// If multiple rules match or none rules match, an error is returned.
func (r *Rules) FindMatchingRule(c context.Context, q *RulesQuery) (*admin.DelegationRule, error) {
	var matches []*admin.DelegationRule
	for _, rule := range r.rules {
		switch yes, err := rule.matchesQuery(c, q); {
		case err != nil:
			return nil, err // usually transient
		case yes:
			matches = append(matches, rule.rule)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matching delegation rules in the config")
	}

	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = fmt.Sprintf("%q", m.Name)
		}
		return nil, fmt.Errorf(
			"ambiguous request, multiple delegation rules match (%s)",
			strings.Join(names, ", "))
	}

	return matches[0], nil
}

// matchesQuery returns true if this rule matches the query.
//
// See doc in config.proto, DelegationRule for exact description of when this
// happens. Basically, all sets in rule must be supersets of corresponding sets
// in RulesQuery.
//
// May return transient errors.
func (rule *delegationRule) matchesQuery(c context.Context, q *RulesQuery) (bool, error) {
	// Rule's 'requestor' set contains the requestor?
	switch found, err := rule.requestors.IsMember(c, q.Requestor); {
	case err != nil:
		return false, err
	case !found:
		return false, nil
	}

	// Rule's 'delegators' set contains the identity being delegated/impersonated?
	switch yes, err := rule.matchesDelegator(c, q); {
	case err != nil:
		return false, err
	case !yes:
		return false, nil
	}

	// Rule's 'audience' is superset of requested audience?
	allowedAudience := rule.audience
	if rule.addRequestorToAudience {
		allowedAudience = identityset.Extend(allowedAudience, q.Requestor)
	}
	if !allowedAudience.IsSuperset(q.Audience) {
		return false, nil
	}

	// Rule's allowed targets is superset of requested targets?
	if !rule.services.IsSuperset(q.Services) {
		return false, nil
	}

	return true, nil
}

// matchesDelegator is true if 'q.Delegator' is in 'delegators' set (logically).
func (rule *delegationRule) matchesDelegator(c context.Context, q *RulesQuery) (bool, error) {
	if rule.addProjectsAsDelegators && q.Delegator.Kind() == identity.Project {
		return true, nil
	}
	allowedDelegators := rule.delegators
	if rule.addRequestorAsDelegator {
		allowedDelegators = identityset.Extend(allowedDelegators, q.Requestor)
	}
	return allowedDelegators.IsMember(c, q.Delegator)
}
