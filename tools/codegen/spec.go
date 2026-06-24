// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResourceSpec is the parsed representation of a codegen YAML spec file.
type ResourceSpec struct {
	Resource      string          `yaml:"resource"`
	GoClientField string          `yaml:"go_client_field"`
	CRUD          CRUDSpec        `yaml:"crud"`
	Attributes    []AttributeSpec `yaml:"attributes"`
}

// CRUDSpec maps Terraform operations to go-kanidm method names.
type CRUDSpec struct {
	Create string `yaml:"create"`
	Read   string `yaml:"read"`
	Update string `yaml:"update"`
	Delete string `yaml:"delete"`
}

// AttributeSpec describes a single Terraform schema attribute.
type AttributeSpec struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"` // string, bool, int64, set_string, map_set_string
	Required  bool   `yaml:"required"`
	Computed  bool   `yaml:"computed"`
	Sensitive bool   `yaml:"sensitive"`
	Optional  bool   `yaml:"optional"`
	Replace   bool   `yaml:"requires_replace"` // triggers ForceNew
}

// ResourceNamePascal returns the resource name in PascalCase (e.g. oauth2_client → OAuth2Client).
func (s *ResourceSpec) ResourceNamePascal() string {
	parts := strings.Split(s.Resource, "_")
	var sb strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return sb.String()
}

// ModelName returns the Go model struct name, e.g. OAuth2ClientResourceModel.
func (s *ResourceSpec) ModelName() string {
	return s.ResourceNamePascal() + "ResourceModel"
}

// ReceiverType returns the private receiver type name, e.g. oauth2ClientResource.
func (s *ResourceSpec) ReceiverType() string {
	return s.Resource + "Resource"
}

// TFTypeName returns the Terraform type string, e.g. "oauth2_client".
func (s *ResourceSpec) TFTypeName() string {
	return s.Resource
}

// loadSpec parses a YAML spec file into a ResourceSpec.
func loadSpec(path string) (*ResourceSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var spec ResourceSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if spec.Resource == "" {
		return nil, fmt.Errorf("spec must set 'resource'")
	}
	if spec.GoClientField == "" {
		return nil, fmt.Errorf("spec must set 'go_client_field'")
	}

	return &spec, nil
}
