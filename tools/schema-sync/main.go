// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

// Command schema-sync detects drift between the Kanidm OpenAPI schema and the
// provider's recorded baseline. It exits non-zero on breaking changes, making
// it suitable as a CI gate.
//
// Usage:
//
//	go run ./tools/schema-sync --url https://idm.example.com --baseline tools/schema-sync/baseline.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

// Baseline represents the set of fields the provider depends on per resource.
type Baseline map[string]ResourceBaseline

// ResourceBaseline lists the fields read and written by the provider for one resource.
type ResourceBaseline struct {
	ReadFields  []string `json:"read_fields"`
	WriteFields []string `json:"write_fields"`
}

// OpenAPISchema is a minimal representation of the OpenAPI document we care about.
type OpenAPISchema struct {
	Components struct {
		Schemas map[string]SchemaObject `json:"schemas"`
	} `json:"components"`
}

// SchemaObject represents a single schema component.
type SchemaObject struct {
	Properties map[string]interface{} `json:"properties"`
}

// DriftReport summarises differences found.
type DriftReport struct {
	Resource string
	New      []string // fields present in live schema, absent from baseline
	Removed  []string // fields present in baseline, absent from live schema
}

func main() {
	kanidmURL := flag.String("url", os.Getenv("KANIDM_URL"), "Base URL of the Kanidm instance")
	baselinePath := flag.String("baseline", "tools/schema-sync/baseline.json", "Path to baseline JSON")
	flag.Parse()

	if *kanidmURL == "" {
		log.Fatal("--url or KANIDM_URL is required")
	}

	baseline, err := loadBaseline(*baselinePath)
	if err != nil {
		log.Fatalf("load baseline: %v", err)
	}

	schema, err := fetchOpenAPISchema(*kanidmURL)
	if err != nil {
		log.Fatalf("fetch schema: %v", err)
	}

	reports := compareBaseline(baseline, schema)

	if len(reports) == 0 {
		fmt.Println("✓ No drift detected.")
		return
	}

	breaking := false
	for _, r := range reports {
		if len(r.Removed) > 0 {
			breaking = true
			fmt.Printf("BREAKING  [%s] fields removed from live schema: %v\n", r.Resource, r.Removed)
		}
		if len(r.New) > 0 {
			fmt.Printf("NEW       [%s] fields added in live schema (potential enhancements): %v\n", r.Resource, r.New)
		}
	}

	if breaking {
		fmt.Fprintln(os.Stderr, "\nBreaking changes detected. Update provider resources and re-run `make schema-diff` to record new baseline.")
		os.Exit(1)
	}
}

func loadBaseline(path string) (Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return b, nil
}

func fetchOpenAPISchema(baseURL string) (*OpenAPISchema, error) {
	url := baseURL + "/v1/openapi.json"
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(url) //nolint:noctx // schema-sync is a CLI tool
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var schema OpenAPISchema
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	return &schema, nil
}

func compareBaseline(baseline Baseline, schema *OpenAPISchema) []DriftReport {
	var reports []DriftReport

	for resource, rb := range baseline {
		// Attempt to find the corresponding schema component.
		// Convention: resource name maps to a PascalCase component, e.g. "person" → "Person".
		componentName := toPascal(resource)
		component, ok := schema.Components.Schemas[componentName]
		if !ok {
			reports = append(reports, DriftReport{
				Resource: resource,
				Removed:  append(rb.ReadFields, rb.WriteFields...),
			})
			continue
		}

		liveFields := make(map[string]bool)
		for k := range component.Properties {
			liveFields[k] = true
		}

		all := unique(append(rb.ReadFields, rb.WriteFields...))

		var removed, added []string
		for _, f := range all {
			if !liveFields[f] {
				removed = append(removed, f)
			}
		}
		for f := range liveFields {
			found := false
			for _, bf := range all {
				if bf == f {
					found = true
					break
				}
			}
			if !found {
				added = append(added, f)
			}
		}

		sort.Strings(removed)
		sort.Strings(added)

		if len(removed) > 0 || len(added) > 0 {
			reports = append(reports, DriftReport{
				Resource: resource,
				Removed:  removed,
				New:      added,
			})
		}
	}

	return reports
}

func toPascal(s string) string {
	if len(s) == 0 {
		return s
	}
	return string([]byte{s[0] - 32}) + s[1:]
}

func unique(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
