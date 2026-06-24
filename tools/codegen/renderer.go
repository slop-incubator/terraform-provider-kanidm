// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// renderer executes named templates from the templates/ directory.
type renderer struct {
	tmplDir string
}

func newRenderer() *renderer {
	// Templates are resolved relative to the binary's working directory.
	// When invoked via `go run ./tools/codegen`, cwd is the module root.
	return &renderer{tmplDir: "tools/codegen/templates"}
}

// render executes the named template with spec and writes to outPath.
// It refuses to overwrite an existing file to prevent accidental data loss.
func (r *renderer) render(tmplName string, spec *ResourceSpec, outPath string) error {
	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("output file %q already exists; delete it first to regenerate", outPath)
	}

	tmplPath := filepath.Join(r.tmplDir, tmplName)
	tmpl, err := template.New(tmplName).
		Funcs(template.FuncMap{
			"tfSchemaType": tfSchemaType,
			"goFieldType":  goFieldType,
		}).
		ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", tmplPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %q: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, tmplName, spec); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// tfSchemaType converts a spec type string to a terraform-plugin-framework schema type.
func tfSchemaType(t string) string {
	switch t {
	case "string":
		return "schema.StringAttribute"
	case "bool":
		return "schema.BoolAttribute"
	case "int64":
		return "schema.Int64Attribute"
	case "set_string":
		return "schema.SetAttribute{ElementType: types.StringType}"
	case "map_set_string":
		return "schema.MapAttribute{ElementType: types.SetType{ElemType: types.StringType}}"
	default:
		return "schema.StringAttribute /* TODO: unknown type " + t + " */"
	}
}

// goFieldType converts a spec type string to the Go tfsdk field type.
func goFieldType(t string) string {
	switch t {
	case "string":
		return "types.String"
	case "bool":
		return "types.Bool"
	case "int64":
		return "types.Int64"
	case "set_string":
		return "types.Set"
	case "map_set_string":
		return "types.Map"
	default:
		return "types.String /* TODO: unknown type " + t + " */"
	}
}
