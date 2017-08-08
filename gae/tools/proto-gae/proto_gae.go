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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/stringsetflag"
)

type app struct {
	out io.Writer

	packageName string
	typeNames   stringsetflag.Flag
	outFile     string
	header      string
}

const help = `Usage of %s:

%s is a go-generator program that generates PropertyConverter implementations
for types produced by protoc. It can be used in a go generation file like:

  //go:generate <protoc command>
  //go:generate proto-gae -type MessageType -type OtherMessageType

This will produce a new file which implements the ToProperty and FromProperty
methods for the named types.

Options:
`

const copyright = `// Copyright 2016 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
`

func (a *app) parseArgs(fs *flag.FlagSet, args []string) error {
	fs.SetOutput(a.out)
	fs.Usage = func() {
		fmt.Fprintf(a.out, help, args[0], args[0])
		fs.PrintDefaults()
	}

	fs.Var(&a.typeNames, "type",
		"A generated proto.Message type to generate stubs for (required, repeatable)")
	fs.StringVar(&a.outFile, "out", "proto_gae.gen.go",
		"The name of the output file")
	fs.StringVar(&a.header, "header", copyright, "Header text to put at the top of "+
		"the generated file. Defaults to the LUCI Authors copyright.")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	fail := errors.MultiError(nil)
	if a.typeNames.Data == nil || a.typeNames.Data.Len() == 0 {
		fail = append(fail, errors.New("must specify one or more -type"))
	}
	if !strings.HasSuffix(a.outFile, ".go") {
		fail = append(fail, errors.New("-output must end with '.go'"))
	}
	if len(fail) > 0 {
		for _, e := range fail {
			fmt.Fprintln(a.out, "error:", e)
		}
		fmt.Fprintln(a.out)
		fs.Usage()
		return fail
	}
	return nil
}

var tmpl = template.Must(
	template.New("main").Parse(`{{if index . "header"}}{{index . "header"}}
{{end}}// AUTOGENERATED: Do not edit

package {{index . "package"}}

import (
	"github.com/golang/protobuf/proto"

	"go.chromium.org/gae/service/datastore"
){{range index . "types"}}

var _ datastore.PropertyConverter = (*{{.}})(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// '{{.}}' to serialize to an unindexed '[]byte' when used with the
// "go.chromium.org/gae" library.
func (p *{{.}}) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded '{{.}}' when used with the "go.chromium.org/gae" library.
func (p *{{.}}) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}{{end}}
`))

func (a *app) writeTo(w io.Writer) error {
	typeNames := a.typeNames.Data.ToSlice()
	sort.Strings(typeNames)

	return tmpl.Execute(w, map[string]interface{}{
		"package": a.packageName,
		"types":   typeNames,
		"header":  a.header,
	})
}

func (a *app) main() {
	if err := a.parseArgs(flag.NewFlagSet(os.Args[0], flag.ContinueOnError), os.Args); err != nil {
		os.Exit(1)
	}
	ofile, err := os.Create(a.outFile)
	if err != nil {
		fmt.Fprintf(a.out, "error: %s", err)
		os.Exit(2)
	}
	closeFn := func(delete bool) {
		if ofile != nil {
			if err := ofile.Close(); err != nil {
				fmt.Fprintf(a.out, "error while closing file: %s", err)
			}
			if delete {
				if err := os.Remove(a.outFile); err != nil {
					fmt.Fprintf(a.out, "failed to remove file!")
				}
			}
		}
		ofile = nil
	}
	defer closeFn(false)
	buf := bufio.NewWriter(ofile)
	err = a.writeTo(buf)
	if err != nil {
		fmt.Fprintf(a.out, "error while writing: %s", err)
		closeFn(true)
		os.Exit(3)
	}
	if err := buf.Flush(); err != nil {
		fmt.Fprintf(a.out, "error while writing: %s", err)
		closeFn(true)
		os.Exit(4)
	}
}

func main() {
	(&app{out: os.Stderr, packageName: os.Getenv("GOPACKAGE")}).main()
}
