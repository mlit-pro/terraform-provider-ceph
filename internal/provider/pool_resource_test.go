/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPoolResource(t *testing.T) {
	const addr = "ceph_pool.test"
	name := acctest.RandomWithPrefix("tf-acc-pool")
	renamed := acctest.RandomWithPrefix("tf-acc-pool")
	var poolID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a replicated pool with one application.
				Config: testAccPoolResourceConfig(name, `["rbd"]`, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "pool_type", "replicated"),
					resource.TestCheckResourceAttr(addr, "pg_num", "8"),
					resource.TestCheckResourceAttrSet(addr, "pool_id"),
					resource.TestCheckResourceAttrSet(addr, "crush_rule"),
					resource.TestCheckResourceAttr(addr, "application_metadata.#", "1"),
					resource.TestCheckTypeSetElemAttr(addr, "application_metadata.*", "rbd"),
					storeAttr(addr, "pool_id", &poolID),
				),
			},
			{
				// Import by name.
				ResourceName:                         addr,
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				// Add an application and set a quota; pool_id must be unchanged.
				Config: testAccPoolResourceConfig(name, `["rbd", "rgw"]`, "  quota_max_bytes = 1073741824"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "quota_max_bytes", "1073741824"),
					resource.TestCheckResourceAttr(addr, "application_metadata.#", "2"),
					resource.TestCheckTypeSetElemAttr(addr, "application_metadata.*", "rgw"),
					resource.TestCheckResourceAttrPtr(addr, "pool_id", &poolID),
				),
			},
			{
				// In-place rename: the pool_id must survive (no destroy/recreate).
				Config: testAccPoolResourceConfig(renamed, `["rbd", "rgw"]`, "  quota_max_bytes = 1073741824"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", renamed),
					resource.TestCheckResourceAttrPtr(addr, "pool_id", &poolID),
				),
			},
			{
				// Clear all applications (empty set must be sent, not omitted).
				Config: testAccPoolResourceConfig(renamed, `[]`, "  quota_max_bytes = 1073741824"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "application_metadata.#", "0"),
					resource.TestCheckResourceAttrPtr(addr, "pool_id", &poolID),
				),
			},
		},
	})
}

func TestAccPoolResourceValidateErasureRequiresProfile(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pool")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "ceph_pool" "test" {
  name      = %q
  pool_type = "erasure"
  pg_num    = 8
}
`, name),
				ExpectError: regexp.MustCompile(`erasure_code_profile is required`),
			},
		},
	})
}

func TestAccPoolResourceValidateReplicatedForbidsProfile(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pool")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "ceph_pool" "test" {
  name                 = %q
  pool_type            = "replicated"
  pg_num               = 8
  erasure_code_profile = "some-profile"
}
`, name),
				ExpectError: regexp.MustCompile(`must not be set`),
			},
		},
	})
}

func testAccPoolResourceConfig(name, applicationMetadata, extra string) string {
	return fmt.Sprintf(`
resource "ceph_pool" "test" {
  name                 = %q
  pool_type            = "replicated"
  pg_num               = 8
  application_metadata = %s
  pg_autoscale_mode    = "off"
%s
}
`, name, applicationMetadata, extra)
}

// storeAttr captures a resource attribute value into dest at check time, so a
// later step can assert it is unchanged (e.g. pool_id across a rename).
func storeAttr(addr, key string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		*dest = rs.Primary.Attributes[key]
		return nil
	}
}
