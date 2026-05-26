/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleResource(t *testing.T) {
	const addr = "ceph_role.test"
	name := acctest.RandomWithPrefix("tf-acc-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResourceConfig(name, `{ pool = ["read"] }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "scopes_permissions.pool.#", "1"),
					resource.TestCheckResourceAttr(addr, "system", "false"),
				),
			},
			{
				ResourceName:                         addr,
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: testAccRoleResourceConfig(name, `{ pool = ["read", "update"], monitor = ["read"] }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "scopes_permissions.pool.#", "2"),
					resource.TestCheckResourceAttr(addr, "scopes_permissions.monitor.#", "1"),
				),
			},
		},
	})
}

func testAccRoleResourceConfig(name, scopes string) string {
	return fmt.Sprintf(`
resource "ceph_role" "test" {
  name               = %q
  scopes_permissions = %s
}
`, name, scopes)
}
