// Copyright 2020 The LUCI Authors.
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

// Command realms-dump can dump realm configs served by an Auth Service.
//
// This is to aid in developing Realms API and debugging issues. Not intended to
// be used in any production setting.
package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
)

var (
	authServiceURL = flag.String("auth-service-url", "https://chrome-infra-auth.appspot.com",
		"https:// URL of a Auth Service to fetch realms from")
)

func main() {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	authFlags := authcli.Flags{}
	authFlags.Register(flag.CommandLine, chromeinfra.DefaultAuthOptions())

	flag.Parse()

	opts, err := authFlags.Options()
	if err != nil {
		return err
	}
	authenticator := auth.NewAuthenticator(ctx, auth.SilentLogin, opts)
	client, err := authenticator.Client()
	if err != nil {
		return err
	}

	authDB, err := fetchAuthDB(ctx, client, *authServiceURL)
	if err != nil {
		return err
	}

	logging.Infof(ctx, "Realms proto:")
	fmt.Printf("%s", proto.MarshalTextString(authDB.Realms))
	return nil
}

func fetchAuthDB(ctx context.Context, client *http.Client, authServiceURL string) (*protocol.AuthDB, error) {
	req, err := http.NewRequest("GET", authServiceURL+"/auth_service/api/v1/authdb/revisions/latest", nil)
	if err != nil {
		return nil, errors.Annotate(err, "failed to prepare the request").Err()
	}

	// Grab JSON with base64-encoded deflated AuthDB snapshot.
	logging.Infof(ctx, "Sending the request to %s...", authServiceURL)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Annotate(err, "failed to send the request to the auth service").Err()
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Annotate(err, "failed to read the response from the auth service").Err()
	}
	if resp.StatusCode != 200 {
		return nil, errors.Reason("unexpected response with code %d from the auth service: %s", resp.StatusCode, body).Err()
	}

	// Extract deflated ReplicationPushRequest from it.
	var out struct {
		Snapshot struct {
			Rev          int64  `json:"auth_db_rev"`
			SHA256       string `json:"sha256"`
			Created      int64  `json:"created_ts"`
			DeflatedBody string `json:"deflated_body"`
		} `json:"snapshot"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, errors.Annotate(err, "failed to JSON unmarshal the response").Err()
	}
	deflated, err := base64.StdEncoding.DecodeString(out.Snapshot.DeflatedBody)
	if err != nil {
		return nil, errors.Annotate(err, "failed to base64-decode").Err()
	}

	// Inflate it.
	reader, err := zlib.NewReader(bytes.NewReader(deflated))
	if err != nil {
		return nil, errors.Annotate(err, "failed to start inflating").Err()
	}
	inflated := bytes.Buffer{}
	if _, err := io.Copy(&inflated, reader); err != nil {
		return nil, errors.Annotate(err, "failed to inflate").Err()
	}
	if err := reader.Close(); err != nil {
		return nil, errors.Annotate(err, "failed to inflate").Err()
	}

	// Unmarshal the actual proto message contained there.
	msg := protocol.ReplicationPushRequest{}
	if err := proto.Unmarshal(inflated.Bytes(), &msg); err != nil {
		return nil, errors.Annotate(err, "failed to deserialize AuthDB proto").Err()
	}

	// Log some stats.
	logging.Infof(ctx, "AuthDB rev %d, created %s by the auth service v%s",
		out.Snapshot.Rev, humanize.Time(time.Unix(0, out.Snapshot.Created*1000)),
		msg.AuthCodeVersion)
	logging.Infof(ctx, "Raw response size: %d bytes", len(body))
	logging.Infof(ctx, "Deflated size:     %d bytes", len(deflated))
	logging.Infof(ctx, "Inflated size:     %d bytes", inflated.Len())

	return msg.AuthDb, nil
}
