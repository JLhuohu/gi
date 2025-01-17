// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

// This program generates constants for all of the icon
// svg file names in outlined
package main

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/icons"
)

const preamble = `// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code generated by go generate (by gen.go); DO NOT EDIT.

package icons

const (`

// iconData contains the data for an icon
type iconData struct {
	Dir   string // Dir is the directory in which the icon is contained
	Snake string // Snake is the snake_case name of the icon
	Camel string // Camel is the CamelCase name of the icon
}

var iconTmpl = template.Must(template.New("icon").Parse(
	`
	// {{.Camel}} is the "{{.Snake}}" icon from Material Design Symbols,
	// defined at https://goki.dev/gi/v2/blob/master/icons/{{.Dir}}{{.Snake}}.svg
	{{.Camel}} Icon = "{{.Snake}}"
	`,
))

func main() {
	buf := bytes.NewBufferString(preamble)

	fs.WalkDir(icons.Icons, "svg", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		dir, name := filepath.Split(path)
		name = strings.TrimSuffix(name, ".svg")
		// ignore fill icons as they are handled separately
		if strings.HasSuffix(name, "-fill") {
			return nil
		}
		camel := strcase.ToCamel(name)
		// identifier names can't start with a digit
		if unicode.IsDigit([]rune(camel)[0]) {
			camel = "X" + camel
		}
		// no backslashes in URL paths
		dir = strings.ReplaceAll(dir, `\`, "/")
		data := iconData{
			Dir:   dir,
			Snake: name,
			Camel: camel,
		}
		return iconTmpl.Execute(buf, data)
	})
	buf.WriteString("\n)")
	err := os.WriteFile("iconnames.go", buf.Bytes(), 0666)
	if err != nil {
		log.Fatalln("error writing result to iconnames.go:", err)
	}
}
