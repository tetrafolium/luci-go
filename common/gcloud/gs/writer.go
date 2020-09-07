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

package gs

import (
	"context"
	"io"
	"time"

	gs "cloud.google.com/go/storage"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
)

// Writer is an augmented io.WriteCloser instance.
type Writer interface {
	io.WriteCloser

	// Count returns the number of bytes written by the object.
	Count() int64
}

type prodWriter struct {
	context.Context

	// Writer is the active Writer instance. It will be nil until the first Write
	// invocation.
	*gs.Writer

	client  *prodClient
	bucket  string
	relpath string
	count   int64
}

var _ Writer = (*prodWriter)(nil)

// Write writes data with exponenital backoff/retry.
func (w *prodWriter) Write(d []byte) (a int, err error) {
	if w.Writer == nil {
		w.Writer = w.client.baseClient.Bucket(w.bucket).Object(w.relpath).NewWriter(w)
	}

	err = retry.Retry(w, transient.Only(retry.Default), func() (ierr error) {
		a, ierr = w.Writer.Write(d)

		// Assume all Write errors are transient.
		ierr = transient.Tag.Apply(ierr)
		return
	}, func(err error, d time.Duration) {
		log.Fields{
			log.ErrorKey: err,
			"delay":      d,
			"bucket":     w.bucket,
			"path":       w.relpath,
		}.Warningf(w, "Transient error on GS write. Retrying...")
	})

	w.count += int64(a)
	return
}

func (w *prodWriter) Close() error {
	if w.Writer == nil {
		return nil
	}

	return retry.Retry(w, transient.Only(retry.Default),
		w.Writer.Close,
		func(err error, d time.Duration) {
			log.Fields{
				log.ErrorKey: err,
				"delay":      d,
				"bucket":     w.bucket,
				"path":       w.relpath,
			}.Warningf(w, "Transient error closing GS Writer. Retrying...")
		})
}

func (w *prodWriter) Count() int64 {
	return w.count
}
