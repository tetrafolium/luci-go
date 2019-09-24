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

package main

import (
	"text/template"

	"go.chromium.org/luci/grpc/internal/svctool"
)

var (
	tmpl = template.Must(template.New("").Parse(
		`// Code generated by svcdec; DO NOT EDIT.

package {{.PackageName}}

import (
	"context"

	proto "github.com/golang/protobuf/proto"

	{{range .ExtraImports}}
	{{.Name}} "{{.Path}}"{{end}}
)

{{range .Services}}
{{$StructName := .StructName}}
type {{$StructName}} struct {
	// Service is the service to decorate.
	Service {{.Service.TypeName}}
	// Prelude is called for each method before forwarding the call to Service.
	// If Prelude returns an error, then the call is skipped and the error is
	// processed via the Postlude (if one is defined), or it is returned directly.
	Prelude func(ctx context.Context, methodName string, req proto.Message) (context.Context, error)
	// Postlude is called for each method after Service has processed the call, or
	// after the Prelude has returned an error. This takes the the Service's
	// response proto (which may be nil) and/or any error. The decorated
	// service will return the response (possibly mutated) and error that Postlude
	// returns.
	Postlude func(ctx context.Context, methodName string, rsp proto.Message, err error) error
}

{{range .Methods}}
func (s *{{$StructName}}) {{.Name}}(ctx context.Context, req {{.InputType}}) (rsp {{.OutputType}}, err error) {
	if s.Prelude != nil {
		var newCtx context.Context
		newCtx, err = s.Prelude(ctx, "{{.Name}}", req)
		if err == nil {
			ctx = newCtx
		}
	}
	if err == nil {
		rsp, err = s.Service.{{.Name}}(ctx, req)
	}
	if s.Postlude != nil {
		err = s.Postlude(ctx, "{{.Name}}", rsp, err)
	}
	return
}
{{end}}
{{end}}
`))
)

type (
	templateArgs struct {
		PackageName  string
		Services     []*service
		ExtraImports []svctool.Import
	}

	service struct {
		*svctool.Service
		StructName string
	}
)
