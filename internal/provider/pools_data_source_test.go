/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPoolsDataSource(t *testing.T) {
	const addr = "data.ceph_pools.all"
	name := acctest.RandomWithPrefix("tf-acc-pool")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolsDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPoolInList(addr, name),
				),
			},
		},
	})
}

// testAccCheckPoolInList asserts that a pool name appears in the pools list of a
// ceph_pools data source.
func testAccCheckPoolInList(addr, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		for k, v := range rs.Primary.Attributes {
			if strings.HasSuffix(k, ".name") && v == name {
				return nil
			}
		}
		return fmt.Errorf("pool %q not found in %s pools list", name, addr)
	}
}

func testAccPoolsDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "ceph_pool" "test" {
  name                 = %q
  pool_type            = "replicated"
  pg_num               = 8
  application_metadata = ["rbd"]
  pg_autoscale_mode    = "off"
}

data "ceph_pools" "all" {
  depends_on = [ceph_pool.test]
}
`, name)
}
