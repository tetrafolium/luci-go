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

// Package validation provides a helper for performing config validations.
package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"
)

// Error is an error with details of validation issues.
//
// Returned by Context.Finalize().
type Error struct {
	// Errors is a list of individual validation errors.
	//
	// Each one is annotated with "file" string, logical path pointing to
	// the element that contains the error, and its severity. It is provided as a
	// slice of strings in "element" annotation.
	Errors errors.MultiError
}

// Error makes *Error implement 'error' interface.
func (e *Error) Error() string {
	return e.Errors.Error()
}

// WithSeverity returns a multi-error with errors of a given severity only.
func (e *Error) WithSeverity(s Severity) error {
	var filtered errors.MultiError
	for _, valErr := range e.Errors {
		if severity, ok := SeverityTag.In(valErr); ok && severity == s {
			filtered = append(filtered, valErr)
		}
	}
	if len(filtered) != 0 {
		return filtered
	}
	return nil
}

// Context is an accumulator for validation errors.
//
// It is passed to a function that does config validation. Such function may
// validate a bunch of files (using SetFile to indicate which one is processed
// now). Each file may have some internal nested structure. The logical path
// inside this structure is captured through Enter and Exit calls.
type Context struct {
	Context context.Context

	errors  errors.MultiError // all accumulated errors, including those with Warning severity.
	file    string            // the currently validated file
	element []string          // logical path of a sub-element we validate, see Enter
}

type fileTagType struct{ Key errors.TagKey }

func (f fileTagType) With(name string) errors.TagValue {
	return errors.TagValue{Key: f.Key, Value: name}
}
func (f fileTagType) In(err error) (v string, ok bool) {
	d, ok := errors.TagValueIn(f.Key, err)
	if ok {
		v = d.(string)
	}
	return
}

type elementTagType struct{ Key errors.TagKey }

func (e elementTagType) With(elements []string) errors.TagValue {
	return errors.TagValue{Key: e.Key, Value: append([]string(nil), elements...)}
}
func (e elementTagType) In(err error) (v []string, ok bool) {
	d, ok := errors.TagValueIn(e.Key, err)
	if ok {
		v = d.([]string)
	}
	return
}

// Severity of the validation message.
//
// Only Blocking and Warning severities are supported.
type Severity int

const (
	// Blocking severity blocks config from being accepted.
	//
	// Corresponds to ValidationResponseMessage_Severity:ERROR.
	Blocking Severity = 0
	// Warning severity doesn't block config from being accepted.
	//
	// Corresponds to ValidationResponseMessage_Severity:WARNING.
	Warning Severity = 1
)

type severityTagType struct{ Key errors.TagKey }

func (s severityTagType) With(severity Severity) errors.TagValue {
	return errors.TagValue{Key: s.Key, Value: severity}
}
func (s severityTagType) In(err error) (v Severity, ok bool) {
	d, ok := errors.TagValueIn(s.Key, err)
	if ok {
		v = d.(Severity)
	}
	return
}

var fileTag = fileTagType{errors.NewTagKey("holds the file name for tests")}
var elementTag = elementTagType{errors.NewTagKey("holds the elements for tests")}

// SeverityTag holds the severity of the given validation error.
var SeverityTag = severityTagType{errors.NewTagKey("holds the severity")}

// Errorf records the given format string and args as a blocking validation error.
func (v *Context) Errorf(format string, args ...interface{}) {
	v.record(Blocking, errors.Reason(format, args...).Err())
}

// Error records the given error as a blocking validation error.
func (v *Context) Error(err error) {
	v.record(Blocking, err)
}

// Warningf records the given format string and args as a validation warning.
func (v *Context) Warningf(format string, args ...interface{}) {
	v.record(Warning, errors.Reason(format, args...).Err())
}

// Warning records the given error as a validation warning.
func (v *Context) Warning(err error) {
	v.record(Warning, err)
}

func (v *Context) record(severity Severity, err error) {
	ctx := ""
	if v.file != "" {
		ctx = fmt.Sprintf("in %q", v.file)
	} else {
		ctx = "in <unspecified file>"
	}
	if len(v.element) != 0 {
		ctx += " (" + strings.Join(v.element, " / ") + ")"
	}
	// Make the file and the logical path also usable through error inspection.
	v.errors = append(v.errors, errors.Annotate(err, "%s", ctx).Tag(
		fileTag.With(v.file), elementTag.With(v.element), SeverityTag.With(severity)).Err())
}

// SetFile records that what follows is errors for this particular file.
//
// Changing the file resets the current element (see Enter/Exit).
func (v *Context) SetFile(path string) {
	if v.file != path {
		v.file = path
		v.element = nil
	}
}

// Enter descends into a sub-element when validating a nested structure.
//
// Useful for defining context. A current path of elements shows up in
// validation messages.
//
// The reverse is Exit.
func (v *Context) Enter(title string, args ...interface{}) {
	e := fmt.Sprintf(title, args...)
	v.element = append(v.element, e)
}

// Exit pops the current element we are visiting from the stack.
//
// This is the reverse of Enter. Each Enter must have corresponding Exit. Use
// functions and defers to ensure this, if it's otherwise hard to track.
func (v *Context) Exit() {
	if len(v.element) != 0 {
		v.element = v.element[:len(v.element)-1]
	}
}

// Finalize returns *Error if some validation errors were recorded.
//
// Returns nil otherwise.
func (v *Context) Finalize() error {
	if len(v.errors) == 0 {
		return nil
	}
	return &Error{
		Errors: append(errors.MultiError{}, v.errors...),
	}
}
