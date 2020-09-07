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

package streamclient

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/clockflag"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/client/butlerlib/streamproto"
	"github.com/tetrafolium/luci-go/logdog/common/types"
)

// This is populated via init() functions in this package.
var protocolRegistry = map[string]dialFactory{}

// dialFactory takes an implementation-specific address (e.g. `localhost:1234`
// or `/path/to/fifo`), and returns a `dialer` function which can be invoked to
// open a new, raw, connection to the butler.
type dialFactory func(addr string) (dialer, error)

// dialer is called by Client for every new stream created, and is expected to:
//   * open a connection (as appropriate) to the dialer's destination
//   * "handshake" with the opened connection (if needed)
//   * return an appropriate writer type around the connection.
type dialer interface {
	// if forProcess is true, this must do its best to return an *os.File.
	//
	// If the implementation fails to do so, then it will cause "os/exec" to
	// allocate an extra goroutine and copy loop when this stream is attached to
	// a command's stdout/stderr.
	DialStream(forProcess bool, f streamproto.Flags) (io.WriteCloser, error)
	DialDgramStream(f streamproto.Flags) (DatagramStream, error)
}

// Client is a client to a local LogDog Butler.
//
// The methods here allow you to open a stream (text, binary or datagram) which
// you can then use to send data to LogDog.
type Client struct {
	dial dialer

	ns types.StreamName
}

// New instantiates a new Client instance. This type of instance will be parsed
// from the supplied path string, which takes the form:
//   <protocol>:<protocol-specific-spec>
//
// Supported protocols
//
// Below is the list of all supported protocols:
//
//   unix:/path/to/socket (POSIX only)
//
// Connects to a UNIX domain socket at "/path/to/socket".
// This is the preferred protocol for Linux/Mac.
//
//   net.pipe:name (Windows only)
//
// Connects to a local Windows named pipe "\\.\pipe\name". This is the preferred
// protocol for Windows.
//
//   null (All platforms)
//
// Sinks all connections and writes into a null data sink. Useful for tests, or
// for running programs which use logdog but you don't care about their logdog
// outputs.
func New(path string, namespace types.StreamName) (*Client, error) {
	parts := strings.SplitN(path, ":", 2)
	protocol, value := parts[0], ""
	if len(parts) == 2 {
		value = parts[1]
	}

	if f, ok := protocolRegistry[protocol]; ok {
		dial, err := f(value)
		if err != nil {
			return nil, errors.Annotate(err, "opening path %q", path).Err()
		}
		return &Client{dial, namespace}, nil
	}
	return nil, errors.Reason("no protocol registered for [%s]", parts[0]).Err()
}

func (c *Client) mkOptions(ctx context.Context, name types.StreamName, sType logpb.StreamType, opts []Option) (ret options, err error) {
	ret.desc.Name = streamproto.StreamNameFlag(c.ns.Concat(name))
	ret.desc.Type = streamproto.StreamType(sType)
	for _, o := range opts {
		o(&ret)
	}
	if time.Time(ret.desc.Timestamp).IsZero() {
		ret.desc.Timestamp = clockflag.Time(clock.Now(ctx))
	}
	if ret.desc.ContentType == "" {
		ret.desc.ContentType = string(ret.desc.Type.DefaultContentType())
	}

	if err = ret.desc.Descriptor().Validate(false); err != nil {
		err = errors.Annotate(err, "invalid stream descriptor").Err()
	}
	return
}

// NewTextStream returns a new open text-based stream to the butler.
//
// Text streams look for newlines to delimit log sections.
func (c *Client) NewTextStream(ctx context.Context, name types.StreamName, opts ...Option) (io.WriteCloser, error) {
	fullOpts, err := c.mkOptions(ctx, name, logpb.StreamType_TEXT, opts)
	if err != nil {
		return nil, err
	}
	ret, err := c.dial.DialStream(fullOpts.forProcess, fullOpts.desc)
	return ret, errors.Annotate(err, "attempting to connect text stream %q", name).Err()
}

// NewBinaryStream returns a new open binary stream to the butler.
//
// Binary streams use fixed size chunks to delimit log sections.
func (c *Client) NewBinaryStream(ctx context.Context, name types.StreamName, opts ...Option) (io.WriteCloser, error) {
	fullOpts, err := c.mkOptions(ctx, name, logpb.StreamType_BINARY, opts)
	if err != nil {
		return nil, err
	}
	ret, err := c.dial.DialStream(fullOpts.forProcess, fullOpts.desc)
	return ret, errors.Annotate(err, "attempting to connect binary stream %q", name).Err()
}

// NewDatagramStream returns a new datagram stream to the butler.
//
// Datagram streams allow you to send messages without having to demark the
// separation between messages.
//
// NOTE: It is an error to pass ForProcess as an Option (see documentation on
// ForProcess for more detail).
func (c *Client) NewDatagramStream(ctx context.Context, name types.StreamName, opts ...Option) (DatagramStream, error) {
	fullOpts, err := c.mkOptions(ctx, name, logpb.StreamType_DATAGRAM, opts)
	if err != nil {
		return nil, err
	}
	if fullOpts.forProcess {
		return nil, errors.Reason("cannot specify ForProcess on a datagram stream").Err()
	}
	ret, err := c.dial.DialDgramStream(fullOpts.desc)
	return ret, errors.Annotate(err, "attempting to connect datagram stream %q", name).Err()
}
