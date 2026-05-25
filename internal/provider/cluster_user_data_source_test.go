/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClusterUserDataSource(t *testing.T) {
	const addr = "data.ceph_cluster_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterUserDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "entity", "client.tf-acc-ds"),
					resource.TestCheckResourceAttr(addr, "capabilities.mon", "allow r"),
					resource.TestCheckResourceAttrSet(addr, "key"),
				),
			},
		},
	})
}

const testAccClusterUserDataSourceConfig = `
resource "ceph_cluster_user" "test" {
  entity       = "client.tf-acc-ds"
  capabilities = { mon = "allow r" }
}

data "ceph_cluster_user" "test" {
  entity = ceph_cluster_user.test.entity
}
`
