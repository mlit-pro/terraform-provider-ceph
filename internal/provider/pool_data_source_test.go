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

func TestAccPoolDataSource(t *testing.T) {
	const addr = "data.ceph_pool.test"
	name := acctest.RandomWithPrefix("tf-acc-pool")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "pool_type", "replicated"),
					resource.TestCheckResourceAttr(addr, "pg_num", "8"),
					resource.TestCheckResourceAttrSet(addr, "pool_id"),
					resource.TestCheckTypeSetElemAttr(addr, "application_metadata.*", "rbd"),
				),
			},
		},
	})
}

func testAccPoolDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "ceph_pool" "test" {
  name                 = %q
  pool_type            = "replicated"
  pg_num               = 8
  application_metadata = ["rbd"]
  pg_autoscale_mode    = "off"
}

data "ceph_pool" "test" {
  name = ceph_pool.test.name
}
`, name)
}
