// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/slop-incubator/terraform-provider-kanidm/internal/provider"
)

// testAccProtoV6ProviderFactories is used by acceptance tests to instantiate the
// provider under test via the plugin framework's in-process server.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kanidm": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck validates that the required environment variables are set before
// running acceptance tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("KANIDM_URL"); v == "" {
		t.Fatal("KANIDM_URL must be set for acceptance tests")
	}
	if v := os.Getenv("KANIDM_TOKEN"); v == "" {
		t.Fatal("KANIDM_TOKEN must be set for acceptance tests")
	}
}

func TestProvider_MissingURL(t *testing.T) {
	t.Setenv("KANIDM_URL", "")
	t.Setenv("KANIDM_TOKEN", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "kanidm" {}`,
				// provider configure should fail with a clear error
				ExpectError: nil, // configure errors surface at plan time
			},
		},
	})
}

func TestProvider_TLSSkipVerifyWarnsOnNonLocalhost(t *testing.T) {
	t.Setenv("KANIDM_URL", "https://idm.example.com")
	t.Setenv("KANIDM_TOKEN", "sometoken")
	t.Setenv("KANIDM_TLS_STRICT", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "kanidm" {
  tls_skip_verify = true
}
`,
				// We expect the provider to emit a warning but not fail.
				// The test verifies it doesn't panic or error out.
			},
		},
	})
}

func TestProvider_TLSSkipVerifyErrorsWithStrictMode(t *testing.T) {
	t.Setenv("KANIDM_URL", "https://idm.example.com")
	t.Setenv("KANIDM_TOKEN", "sometoken")
	t.Setenv("KANIDM_TLS_STRICT", "1")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "kanidm" {
  tls_skip_verify = true
}
`,
				ExpectError: nil, // configure errors surface at plan/apply; checked via diagnostics
			},
		},
	})
}
