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

package logdog

import (
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetMessageProject implements ProjectBoundMessage.
func (ar *RegisterStreamRequest) GetMessageProject() string { return ar.Project }

// GetMessageProject implements ProjectBoundMessage.
func (ar *LoadStreamRequest) GetMessageProject() string { return ar.Project }

// GetMessageProject implements ProjectBoundMessage.
func (ar *TerminateStreamRequest) GetMessageProject() string { return ar.Project }

// GetMessageProject implements ProjectBoundMessage.
func (ar *ArchiveStreamRequest) GetMessageProject() string { return ar.Project }

// Complete returns true if the archive request expresses that the archived
// log stream was complete.
//
// A log stream is complete if every entry between zero and its terminal index
// is included.
func (ar *ArchiveStreamRequest) Complete() bool {
	tidx := ar.TerminalIndex
	if tidx < 0 {
		tidx = -1
	}
	return (ar.LogEntryCount == (tidx + 1))
}

// MakeError returns an Error object for err. If err is a wrapped gRPC error,
// its code will be extracted and embedded in the returned Error.
//
// The Msg field will not be populated.
func MakeError(err error) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		GrpcCode:  int32(grpcutil.Code(err)),
		Transient: transient.Tag.In(err),
	}
}

// ToError converts an Error into a gRPC error. If e is nil, a nil error will
// be returned.
func (e *Error) ToError() error {
	if e == nil {
		return nil
	}

	code := codes.Code(e.GrpcCode)
	err := status.Errorf(code, "%s", e.Msg)
	if e.Transient || grpcutil.IsTransientCode(code) {
		err = transient.Tag.Apply(err)
	}
	return err
}
