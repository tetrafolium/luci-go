// Copyright 2017 The LUCI Authors.
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
	"net"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/bqlog"
	"github.com/tetrafolium/luci-go/common/bq"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	bqpb "github.com/tetrafolium/luci-go/tokenserver/api/bq"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils"
)

var delegationTokensLog = bqlog.Log{
	QueueName:           "bqlog-delegation-tokens", // see queues.yaml
	DatasetID:           "tokens",                  // see bq/README.md
	TableID:             "delegation_tokens",       // see bq/tables/delegation_tokens.schema
	DumpEntriesToLogger: true,
	DryRun:              appengine.IsDevAppServer(),
}

// MintedTokenInfo is passed to LogToken.
//
// It carries all information about the token minting operation and the produced
// token.
type MintedTokenInfo struct {
	Request   *minter.MintDelegationTokenRequest  // RPC input, as is
	Response  *minter.MintDelegationTokenResponse // RPC output, as is
	ConfigRev string                              // revision of the delegation.cfg used
	Rule      *admin.DelegationRule               // the particular rule used to authorize the request
	PeerIP    net.IP                              // caller IP address
	RequestID string                              // GAE request ID that handled the RPC
	AuthDBRev int64                               // revision of groups database (or 0 if unknown)
}

// toBigQueryMessage returns a message to upload to BigQuery.
func (i *MintedTokenInfo) toBigQueryMessage() *bqpb.DelegationToken {
	subtok := i.Response.DelegationSubtoken
	return &bqpb.DelegationToken{
		// Information about the produced token.
		Fingerprint:       utils.TokenFingerprint(i.Response.Token),
		TokenKind:         subtok.Kind,
		TokenId:           fmt.Sprintf("%d", subtok.SubtokenId),
		DelegatedIdentity: subtok.DelegatedIdentity,
		RequestorIdentity: subtok.RequestorIdentity,
		IssuedAt:          &timestamp.Timestamp{Seconds: subtok.CreationTime},
		Expiration:        &timestamp.Timestamp{Seconds: subtok.CreationTime + int64(subtok.ValidityDuration)},
		TargetAudience:    subtok.Audience,
		TargetServices:    subtok.Services,

		// Information about the request.
		RequestedValidity: i.Request.ValidityDuration,
		RequestedIntent:   i.Request.Intent,
		Tags:              subtok.Tags,

		// Information about the delegation rule used.
		ConfigRev:  i.ConfigRev,
		ConfigRule: i.Rule.Name,

		// Information about the request handler environment.
		PeerIp:         i.PeerIP.String(),
		ServiceVersion: i.Response.ServiceVersion,
		GaeRequestId:   i.RequestID,
		AuthDbRev:      i.AuthDBRev,
	}
}

// LogToken records information about the token in the BigQuery.
//
// The signed token itself is not logged. Only first 16 bytes of its SHA256 hash
// (aka 'fingerprint') is. It is used only to identify this particular token in
// logs.
//
// On dev server, logs to the GAE log only, not to BigQuery (to avoid
// accidentally pushing fake data to real BigQuery dataset).
func LogToken(c context.Context, i *MintedTokenInfo) error {
	return delegationTokensLog.Insert(c, &bq.Row{
		Message: i.toBigQueryMessage(),
	})
}

// FlushTokenLog sends all buffered logged tokens to BigQuery.
//
// It is fine to call FlushTokenLog concurrently from multiple request handlers,
// if necessary (it will effectively parallelize the flush).
func FlushTokenLog(c context.Context) error {
	_, err := delegationTokensLog.Flush(c)
	return err
}
