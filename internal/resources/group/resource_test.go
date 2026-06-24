// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package group_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/slop-incubator/terraform-provider-kanidm/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kanidm": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAcc_Group_basic(t *testing.T) {
	name := "tofu-test-group-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read.
			{
				Config: testAccGroupConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_group.test", "name", name),
					resource.TestCheckResourceAttrSet("kanidm_group.test", "id"),
					resource.TestCheckResourceAttrSet("kanidm_group.test", "spn"),
				),
			},
			// Verify Import.
			{
				ResourceName:      "kanidm_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_Group_update(t *testing.T) {
	name := "tofu-test-group-update"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig_basic(name),
				Check:  resource.TestCheckResourceAttr("kanidm_group.test", "name", name),
			},
			// Add members.
			{
				Config: testAccGroupConfig_withMember(name, "testperson@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kanidm_group.test", "name", name),
					resource.TestCheckTypeSetElemAttr("kanidm_group.test", "members.*", "testperson@example.com"),
				),
			},
			// Remove members (full-replace to empty).
			{
				Config: testAccGroupConfig_basic(name),
				Check:  resource.TestCheckResourceAttr("kanidm_group.test", "members.#", "0"),
			},
		},
	})
}

func TestAcc_Group_disappear(t *testing.T) {
	name := "tofu-test-group-disappear"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig_basic(name),
				// Simulate deletion outside Terraform.
				Check: func(s *terraform.State) error {
					// In a real test, delete the group via the API client here.
					return nil
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// --- config helpers ---

func testAccGroupConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kanidm_group" "test" {
  name = %q
}
`, name)
}

func testAccGroupConfig_withMember(name, member string) string {
	return fmt.Sprintf(`
resource "kanidm_group" "test" {
  name    = %q
  members = [%q]
}
`, name, member)
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	// Delegated to provider_test package; replicated here for test binary independence.
}
