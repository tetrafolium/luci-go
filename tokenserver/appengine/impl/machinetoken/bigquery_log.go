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

package machinetoken

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/bqlog"
	"github.com/tetrafolium/luci-go/common/bq"

	"github.com/tetrafolium/luci-go/tokenserver/api"
	bqpb "github.com/tetrafolium/luci-go/tokenserver/api/bq"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/certconfig"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils"
)

var machineTokensLog = bqlog.Log{
	QueueName:           "bqlog-machine-tokens", // see queues.yaml
	DatasetID:           "tokens",               // see bq/README.md
	TableID:             "machine_tokens",       // see bq/tables/machine_tokens.schema
	DumpEntriesToLogger: true,
	DryRun:              appengine.IsDevAppServer(),
}

// MintedTokenInfo is passed to LogToken.
//
// It carries all information about the token minting operation and the produced
// token.
type MintedTokenInfo struct {
	Request   *minter.MachineTokenRequest   // the token request, as presented by the client
	Response  *minter.MachineTokenResponse  // the response, as returned by the minter
	TokenBody *tokenserver.MachineTokenBody // deserialized token (same as in Response)
	CA        *certconfig.CA                // CA configuration used to authorize this request
	PeerIP    net.IP                        // caller IP address
	RequestID string                        // GAE request ID that handled the RPC
}

// toBigQueryMessage returns a message to upload to BigQuery.
func (i *MintedTokenInfo) toBigQueryMessage() *bqpb.MachineToken {
	// LUCI_MACHINE_TOKEN is the only supported type currently.
	if i.Request.TokenType != tokenserver.MachineTokenType_LUCI_MACHINE_TOKEN {
		panic("unknown token type")
	}
	return &bqpb.MachineToken{
		// Identifier of the token body.
		Fingerprint: utils.TokenFingerprint(i.Response.GetLuciMachineToken().MachineToken),

		// Information about the token.
		MachineFqdn:        i.TokenBody.MachineFqdn,
		TokenType:          i.Request.TokenType,
		IssuedAt:           &timestamp.Timestamp{Seconds: int64(i.TokenBody.IssuedAt)},
		Expiration:         &timestamp.Timestamp{Seconds: int64(i.TokenBody.IssuedAt + i.TokenBody.Lifetime)},
		CertSerialNumber:   fmt.Sprintf("%d", i.TokenBody.CertSn),
		SignatureAlgorithm: i.Request.SignatureAlgorithm,

		// Information about the CA used to authorize this request.
		CaCommonName: i.CA.CN,
		CaConfigRev:  i.CA.UpdatedRev,

		// Information about the request handler.
		PeerIp:         i.PeerIP.String(),
		ServiceVersion: i.Response.ServiceVersion,
		GaeRequestId:   i.RequestID,
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
	return machineTokensLog.Insert(c, &bq.Row{
		Message: i.toBigQueryMessage(),
	})
}

// FlushTokenLog sends all buffered logged tokens to BigQuery.
//
// It is fine to call FlushTokenLog concurrently from multiple request handlers,
// if necessary (it will effectively parallelize the flush).
func FlushTokenLog(c context.Context) error {
	_, err := machineTokensLog.Flush(c)
	return err
}
