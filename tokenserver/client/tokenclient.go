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

package client

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"
)

// Client can make signed requests to the token server.
type Client struct {
	// Client is interface to use for raw RPC calls to the token server.
	//
	// Use minter.NewTokenMinterClient (or NewTokenMinterPRPCClient) to
	// create it. Note that transport-level authentication is not needed.
	Client TokenMinterClient

	// Signer knows how to sign requests using some private key.
	Signer Signer
}

// TokenMinterClient is subset of minter.TokenMinterClient this package uses.
type TokenMinterClient interface {
	// MintMachineToken generates a new token for an authenticated machine.
	MintMachineToken(context.Context, *minter.MintMachineTokenRequest, ...grpc.CallOption) (*minter.MintMachineTokenResponse, error)
}

// Signer knows how to sign requests using some private key.
type Signer interface {
	// Algo returns an algorithm that the signer implements.
	Algo(ctx context.Context) (x509.SignatureAlgorithm, error)

	// Certificate returns ASN.1 DER blob with the certificate of the signer.
	Certificate(ctx context.Context) ([]byte, error)

	// Sign signs a blob using the private key.
	Sign(ctx context.Context, blob []byte) ([]byte, error)
}

// RPCError is optionally returned for recognized RPC errors.
//
// Use typecast to distinguish recognized and unrecognized errors.
type RPCError struct {
	error

	GrpcCode       codes.Code       // grpc-level status code
	ErrorCode      minter.ErrorCode // protocol-level status code
	ServiceVersion string           // version of the backend, if known
}

// MintMachineToken signs the request using the signer and sends it.
//
// It will update in-place the following fields of the request:
//   * Certificate will be set to ASN1 cert corresponding to the signer key.
//   * SignatureAlgorithm will be set to the algorithm used to sign the request.
//   * IssuedAt will be set to the current time.
//
// The rest of the fields must be already populated by the caller and will be
// sent to the server as is.
//
// Returns:
//   * TokenResponse on success.
//   * Non-transient error on fatal errors.
//   * Transient error on transient errors.
//
// You can sniff error for RPCError type to grab more error details.
func (c *Client) MintMachineToken(ctx context.Context, req *minter.MachineTokenRequest, opts ...grpc.CallOption) (*minter.MachineTokenResponse, error) {
	// Fill in SignatureAlgorithm.
	algo, err := c.Signer.Algo(ctx)
	if err != nil {
		return nil, err
	}
	switch algo {
	case x509.SHA256WithRSA:
		req.SignatureAlgorithm = minter.SignatureAlgorithm_SHA256_RSA_ALGO
	default:
		return nil, fmt.Errorf("unsupported signing algorithm - %s", algo)
	}

	// Fill in Certificate and IssuedAt.
	if req.Certificate, err = c.Signer.Certificate(ctx); err != nil {
		return nil, err
	}
	req.IssuedAt = google.NewTimestamp(clock.Now(ctx))

	// Serialize and sign.
	tokenRequest, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	signature, err := c.Signer.Sign(ctx, tokenRequest)
	if err != nil {
		return nil, err
	}

	// Make an RPC call (with retries done by pRPC client).
	resp, err := c.Client.MintMachineToken(ctx, &minter.MintMachineTokenRequest{
		SerializedTokenRequest: tokenRequest,
		Signature:              signature,
	}, opts...)

	// Fatal pRPC-level error or transient error in case retries didn't help.
	if err != nil {
		code := grpc.Code(err)
		err = RPCError{
			error:    err,
			GrpcCode: code,
		}
		if grpcutil.IsTransientCode(code) {
			err = transient.Tag.Apply(err)
		}
		return nil, err
	}

	// The response still may indicate a fatal error.
	if resp.ErrorCode != minter.ErrorCode_SUCCESS {
		details := resp.ErrorMessage
		if details == "" {
			details = "no detailed error message"
		}
		return nil, RPCError{
			error:          fmt.Errorf("token server error %s - %s", resp.ErrorCode, details),
			GrpcCode:       codes.OK,
			ErrorCode:      resp.ErrorCode,
			ServiceVersion: resp.ServiceVersion,
		}
	}

	// Must not happen. But better return an error than nil-panic if it does.
	if resp.TokenResponse == nil {
		return nil, fmt.Errorf("token server didn't return a token")
	}

	return resp.TokenResponse, nil
}
