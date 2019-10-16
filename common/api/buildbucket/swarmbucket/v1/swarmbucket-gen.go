// Copyright 2019 The LUCI Authors.
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

// Code generated file. DO NOT EDIT.

// Package swarmbucket provides access to the Buildbucket-Swarming integration.
//
// Creating a client
//
// Usage example:
//
//   import "go.chromium.org/luci/common/api/buildbucket/swarmbucket/v1"
//   ...
//   ctx := context.Background()
//   swarmbucketService, err := swarmbucket.NewService(ctx)
//
// In this example, Google Application Default Credentials are used for authentication.
//
// For information on how to create and obtain Application Default Credentials, see https://developers.google.com/identity/protocols/application-default-credentials.
//
// Other authentication options
//
// To use an API key for authentication (note: some APIs do not support API keys), use option.WithAPIKey:
//
//   swarmbucketService, err := swarmbucket.NewService(ctx, option.WithAPIKey("AIza..."))
//
// To use an OAuth token (e.g., a user token obtained via a three-legged OAuth flow), use option.WithTokenSource:
//
//   config := &oauth2.Config{...}
//   // ...
//   token, err := config.Exchange(ctx, ...)
//   swarmbucketService, err := swarmbucket.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
//
// See https://godoc.org/google.golang.org/api/option/ for details on options.
package swarmbucket // import "go.chromium.org/luci/common/api/buildbucket/swarmbucket/v1"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	googleapi "google.golang.org/api/googleapi"
	gensupport "go.chromium.org/luci/common/api/internal/gensupport"
	option "google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = gensupport.MarshalJSON
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace
var _ = context.Canceled

const apiId = "swarmbucket:v1"
const apiName = "swarmbucket"
const apiVersion = "v1"
const basePath = "http://localhost:8080/_ah/api/swarmbucket/v1"

// OAuth2 scopes used by this API.
const (
	// https://www.googleapis.com/auth/userinfo.email
	UserinfoEmailScope = "https://www.googleapis.com/auth/userinfo.email"
)

// NewService creates a new Service.
func NewService(ctx context.Context, opts ...option.ClientOption) (*Service, error) {
	scopesOption := option.WithScopes(
		"https://www.googleapis.com/auth/userinfo.email",
	)
	// NOTE: prepend, so we don't override user-specified scopes.
	opts = append([]option.ClientOption{scopesOption}, opts...)
	client, endpoint, err := htransport.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	s, err := New(client)
	if err != nil {
		return nil, err
	}
	if endpoint != "" {
		s.BasePath = endpoint
	}
	return s, nil
}

// New creates a new Service. It uses the provided http.Client for requests.
//
// Deprecated: please use NewService instead.
// To provide a custom HTTP client, use option.WithHTTPClient.
// If you are using google.golang.org/api/googleapis/transport.APIKey, use option.WithAPIKey with NewService instead.
func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	return s, nil
}

type Service struct {
	client    *http.Client
	BasePath  string // API endpoint base URL
	UserAgent string // optional additional User-Agent fragment
}

func (s *Service) userAgent() string {
	if s.UserAgent == "" {
		return googleapi.UserAgent
	}
	return googleapi.UserAgent + " " + s.UserAgent
}

type LegacyApiPubSubCallbackMessage struct {
	AuthToken string `json:"auth_token,omitempty"`

	Topic string `json:"topic,omitempty"`

	UserData string `json:"user_data,omitempty"`

	// ForceSendFields is a list of field names (e.g. "AuthToken") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AuthToken") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacyApiPubSubCallbackMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacyApiPubSubCallbackMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacyApiPutRequestMessage struct {
	Bucket string `json:"bucket,omitempty"`

	// Possible values:
	//   "AUTO"
	//   "CANARY"
	//   "PROD"
	CanaryPreference string `json:"canary_preference,omitempty"`

	ClientOperationId string `json:"client_operation_id,omitempty"`

	Experimental bool `json:"experimental,omitempty"`

	LeaseExpirationTs int64 `json:"lease_expiration_ts,omitempty,string"`

	ParametersJson string `json:"parameters_json,omitempty"`

	PubsubCallback *LegacyApiPubSubCallbackMessage `json:"pubsub_callback,omitempty"`

	Tags []string `json:"tags,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Bucket") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Bucket") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacyApiPutRequestMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacyApiPutRequestMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiBucketMessage struct {
	Builders []*LegacySwarmbucketApiBuilderMessage `json:"builders,omitempty"`

	Name string `json:"name,omitempty"`

	SwarmingHostname string `json:"swarming_hostname,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Builders") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Builders") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiBucketMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiBucketMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiBuilderMessage struct {
	Category string `json:"category,omitempty"`

	Name string `json:"name,omitempty"`

	PropertiesJson string `json:"properties_json,omitempty"`

	SwarmingDimensions []string `json:"swarming_dimensions,omitempty"`

	SwarmingHostname string `json:"swarming_hostname,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Category") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Category") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiBuilderMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiBuilderMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiGetBuildersResponseMessage struct {
	Buckets []*LegacySwarmbucketApiBucketMessage `json:"buckets,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Buckets") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Buckets") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiGetBuildersResponseMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiGetBuildersResponseMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiGetTaskDefinitionRequestMessage struct {
	BuildRequest *LegacyApiPutRequestMessage `json:"build_request,omitempty"`

	// ForceSendFields is a list of field names (e.g. "BuildRequest") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "BuildRequest") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiGetTaskDefinitionRequestMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiGetTaskDefinitionRequestMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiGetTaskDefinitionResponseMessage struct {
	SwarmingHost string `json:"swarming_host,omitempty"`

	TaskDefinition string `json:"task_definition,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "SwarmingHost") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "SwarmingHost") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiGetTaskDefinitionResponseMessage) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiGetTaskDefinitionResponseMessage
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

type LegacySwarmbucketApiSetNextBuildNumberRequest struct {
	Bucket string `json:"bucket,omitempty"`

	Builder string `json:"builder,omitempty"`

	NextNumber int64 `json:"next_number,omitempty,string"`

	// ForceSendFields is a list of field names (e.g. "Bucket") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Bucket") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LegacySwarmbucketApiSetNextBuildNumberRequest) MarshalJSON() ([]byte, error) {
	type NoMethod LegacySwarmbucketApiSetNextBuildNumberRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// method id "swarmbucket.get_builders":

type GetBuildersCall struct {
	s            *Service
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// GetBuilders: Returns defined swarmbucket builders. Returns legacy
// bucket names, e.g. "luci.chromium.try", not "chromium/try". Can be
// used to discover builders.
func (s *Service) GetBuilders() *GetBuildersCall {
	c := &GetBuildersCall{s: s, urlParams_: make(gensupport.URLParams)}
	return c
}

// Bucket sets the optional parameter "bucket":
func (c *GetBuildersCall) Bucket(bucket ...string) *GetBuildersCall {
	c.urlParams_.SetMulti("bucket", append([]string{}, bucket...))
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *GetBuildersCall) Fields(s ...googleapi.Field) *GetBuildersCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *GetBuildersCall) IfNoneMatch(entityTag string) *GetBuildersCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *GetBuildersCall) Context(ctx context.Context) *GetBuildersCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *GetBuildersCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *GetBuildersCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	reqHeaders.Set("x-goog-api-client", "gl-go/1.13.0 gdcl/20191012")
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "builders")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("GET", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "swarmbucket.get_builders" call.
// Exactly one of *LegacySwarmbucketApiGetBuildersResponseMessage or
// error will be non-nil. Any non-2xx status code is an error. Response
// headers are in either
// *LegacySwarmbucketApiGetBuildersResponseMessage.ServerResponse.Header
// or (if a response was returned at all) in
// error.(*googleapi.Error).Header. Use googleapi.IsNotModified to check
// whether the returned error was because http.StatusNotModified was
// returned.
func (c *GetBuildersCall) Do(opts ...googleapi.CallOption) (*LegacySwarmbucketApiGetBuildersResponseMessage, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &LegacySwarmbucketApiGetBuildersResponseMessage{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns defined swarmbucket builders. Returns legacy bucket names, e.g. \"luci.chromium.try\", not \"chromium/try\". Can be used to discover builders.",
	//   "httpMethod": "GET",
	//   "id": "swarmbucket.get_builders",
	//   "parameters": {
	//     "bucket": {
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "builders",
	//   "response": {
	//     "$ref": "LegacySwarmbucketApiGetBuildersResponseMessage"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/userinfo.email"
	//   ]
	// }

}

// method id "swarmbucket.get_task_def":

type GetTaskDefCall struct {
	s                                                   *Service
	legacyswarmbucketapigettaskdefinitionrequestmessage *LegacySwarmbucketApiGetTaskDefinitionRequestMessage
	urlParams_                                          gensupport.URLParams
	ctx_                                                context.Context
	header_                                             http.Header
}

// GetTaskDef: Returns a swarming task definition for a build request.
func (s *Service) GetTaskDef(legacyswarmbucketapigettaskdefinitionrequestmessage *LegacySwarmbucketApiGetTaskDefinitionRequestMessage) *GetTaskDefCall {
	c := &GetTaskDefCall{s: s, urlParams_: make(gensupport.URLParams)}
	c.legacyswarmbucketapigettaskdefinitionrequestmessage = legacyswarmbucketapigettaskdefinitionrequestmessage
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *GetTaskDefCall) Fields(s ...googleapi.Field) *GetTaskDefCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *GetTaskDefCall) Context(ctx context.Context) *GetTaskDefCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *GetTaskDefCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *GetTaskDefCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	reqHeaders.Set("x-goog-api-client", "gl-go/1.13.0 gdcl/20191012")
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.legacyswarmbucketapigettaskdefinitionrequestmessage)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "get_task_def")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("POST", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "swarmbucket.get_task_def" call.
// Exactly one of *LegacySwarmbucketApiGetTaskDefinitionResponseMessage
// or error will be non-nil. Any non-2xx status code is an error.
// Response headers are in either
// *LegacySwarmbucketApiGetTaskDefinitionResponseMessage.ServerResponse.H
// eader or (if a response was returned at all) in
// error.(*googleapi.Error).Header. Use googleapi.IsNotModified to check
// whether the returned error was because http.StatusNotModified was
// returned.
func (c *GetTaskDefCall) Do(opts ...googleapi.CallOption) (*LegacySwarmbucketApiGetTaskDefinitionResponseMessage, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &LegacySwarmbucketApiGetTaskDefinitionResponseMessage{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns a swarming task definition for a build request.",
	//   "httpMethod": "POST",
	//   "id": "swarmbucket.get_task_def",
	//   "path": "get_task_def",
	//   "request": {
	//     "$ref": "LegacySwarmbucketApiGetTaskDefinitionRequestMessage",
	//     "parameterName": "resource"
	//   },
	//   "response": {
	//     "$ref": "LegacySwarmbucketApiGetTaskDefinitionResponseMessage"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/userinfo.email"
	//   ]
	// }

}

// method id "swarmbucket.set_next_build_number":

type SetNextBuildNumberCall struct {
	s                                             *Service
	legacyswarmbucketapisetnextbuildnumberrequest *LegacySwarmbucketApiSetNextBuildNumberRequest
	urlParams_                                    gensupport.URLParams
	ctx_                                          context.Context
	header_                                       http.Header
}

// SetNextBuildNumber: Sets the build number that will be used for the
// next build.
func (s *Service) SetNextBuildNumber(legacyswarmbucketapisetnextbuildnumberrequest *LegacySwarmbucketApiSetNextBuildNumberRequest) *SetNextBuildNumberCall {
	c := &SetNextBuildNumberCall{s: s, urlParams_: make(gensupport.URLParams)}
	c.legacyswarmbucketapisetnextbuildnumberrequest = legacyswarmbucketapisetnextbuildnumberrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SetNextBuildNumberCall) Fields(s ...googleapi.Field) *SetNextBuildNumberCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *SetNextBuildNumberCall) Context(ctx context.Context) *SetNextBuildNumberCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *SetNextBuildNumberCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *SetNextBuildNumberCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	reqHeaders.Set("x-goog-api-client", "gl-go/1.13.0 gdcl/20191012")
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.legacyswarmbucketapisetnextbuildnumberrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "set_next_build_number")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("POST", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "swarmbucket.set_next_build_number" call.
func (c *SetNextBuildNumberCall) Do(opts ...googleapi.CallOption) error {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Sets the build number that will be used for the next build.",
	//   "httpMethod": "POST",
	//   "id": "swarmbucket.set_next_build_number",
	//   "path": "set_next_build_number",
	//   "request": {
	//     "$ref": "LegacySwarmbucketApiSetNextBuildNumberRequest",
	//     "parameterName": "resource"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/userinfo.email"
	//   ]
	// }

}
