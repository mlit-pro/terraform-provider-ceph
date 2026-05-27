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

func TestAccUserResource(t *testing.T) {
	const addr = "ceph_user.test"
	username := acctest.RandomWithPrefix("tf-acc-user")
	role := acctest.RandomWithPrefix("tf-acc-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig(role, username, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "username", username),
					resource.TestCheckResourceAttr(addr, "enabled", "true"),
					resource.TestCheckResourceAttr(addr, "roles.#", "1"),
					resource.TestCheckResourceAttrSet(addr, "last_update"),
				),
			},
			{
				ResourceName:                         addr,
				ImportState:                          true,
				ImportStateId:                        username,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "username",
				ImportStateVerifyIgnore:              []string{"password"},
			},
			{
				Config: testAccUserResourceConfig(role, username, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "enabled", "false"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(role, username string, enabled bool) string {
	return fmt.Sprintf(`
resource "ceph_role" "test" {
  name = %q
  scopes_permissions = {
    pool = ["read"]
  }
}

resource "ceph_user" "test" {
  username = %q
  password = "P@ssw0rd-acc-test"
  roles    = [ceph_role.test.name]
  enabled  = %t
}
`, role, username, enabled)
}
