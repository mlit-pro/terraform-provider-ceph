/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClusterUserResource(t *testing.T) {
	const addr = "ceph_cluster_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterUserResourceConfig(`{ mon = "allow r" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "entity", "client.tf-acc-test"),
					resource.TestCheckResourceAttr(addr, "capabilities.mon", "allow r"),
					resource.TestCheckResourceAttrSet(addr, "key"),
					resource.TestCheckResourceAttrSet(addr, "keyring"),
				),
			},
			{
				ResourceName:                         addr,
				ImportState:                          true,
				ImportStateId:                        "client.tf-acc-test",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "entity",
				ImportStateVerifyIgnore:              []string{"keyring"},
			},
			{
				Config: testAccClusterUserResourceConfig(`{ mon = "allow r", osd = "allow rwx pool=rbd" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "capabilities.mon", "allow r"),
					resource.TestCheckResourceAttr(addr, "capabilities.osd", "allow rwx pool=rbd"),
				),
			},
		},
	})
}

func testAccClusterUserResourceConfig(capabilities string) string {
	return fmt.Sprintf(`
resource "ceph_cluster_user" "test" {
  entity       = "client.tf-acc-test"
  capabilities = %s
}
`, capabilities)
}
