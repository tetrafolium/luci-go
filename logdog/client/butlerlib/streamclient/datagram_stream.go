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

package streamclient

import (
	"io"

	"github.com/tetrafolium/luci-go/common/data/recordio"
)

// DatagramStream is the interface for datagram oriented streams.
type DatagramStream interface {
	io.Closer

	// WriteDatagram writes `dg` as a single datagram.
	WriteDatagram(dg []byte) error
}

// datagramStreamWriter implements a recordio.Frame-based DatagramStream on top
// of a raw writer.
//
// NOTE: Writing to the underlying stream directly will corrupt the datagram
// stream. Don't do that :).
type datagramStreamWriter struct {
	Raw io.WriteCloser
}

// Close implements io.Closer.
func (w *datagramStreamWriter) Close() error {
	return w.Raw.Close()
}

// WriteDatagram implements DatagramStream.
func (w *datagramStreamWriter) WriteDatagram(dg []byte) error {
	_, err := recordio.WriteFrame(w.Raw, dg)
	return err
}
