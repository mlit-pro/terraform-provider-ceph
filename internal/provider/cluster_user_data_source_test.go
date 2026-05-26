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

func TestAccClusterUserDataSource(t *testing.T) {
	const addr = "data.ceph_cluster_user.test"
	entity := acctest.RandomWithPrefix("client.tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterUserDataSourceConfig(entity),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "entity", entity),
					resource.TestCheckResourceAttr(addr, "capabilities.mon", "allow r"),
					resource.TestCheckResourceAttrSet(addr, "key"),
				),
			},
		},
	})
}

func testAccClusterUserDataSourceConfig(entity string) string {
	return fmt.Sprintf(`
resource "ceph_cluster_user" "test" {
  entity       = %q
  capabilities = { mon = "allow r" }
}

data "ceph_cluster_user" "test" {
  entity = ceph_cluster_user.test.entity
}
`, entity)
}
