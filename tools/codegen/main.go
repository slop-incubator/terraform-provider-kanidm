// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

// Command codegen scaffolds a new provider resource or data source from a YAML spec.
//
// Usage:
//
//	go run ./tools/codegen --spec tools/codegen/specs/my_resource.yaml --out internal/resources/my_resource/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	specPath := flag.String("spec", "", "Path to the resource spec YAML file (required)")
	outDir := flag.String("out", "", "Output directory for generated files (required)")
	flag.Parse()

	if *specPath == "" || *outDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	spec, err := loadSpec(*specPath)
	if err != nil {
		log.Fatalf("failed to load spec %q: %v", *specPath, err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory %q: %v", *outDir, err)
	}

	r := newRenderer()

	files := map[string]string{
		"resource.go":      "resource.go.tmpl",
		"resource_test.go": "resource_test.go.tmpl",
	}

	for outFile, tmplName := range files {
		outPath := filepath.Join(*outDir, outFile)
		if err := r.render(tmplName, spec, outPath); err != nil {
			log.Fatalf("failed to render %q: %v", outPath, err)
		}
		fmt.Printf("wrote %s\n", outPath)
	}

	fmt.Println("\nScaffold complete. Review TODO markers in generated files before use.")
}
