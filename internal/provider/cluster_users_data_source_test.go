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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClusterUsersDataSource(t *testing.T) {
	const addr = "data.ceph_cluster_users.all"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterUsersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUserInList(addr, "client.tf-acc-list"),
				),
			},
		},
	})
}

// testAccCheckUserInList asserts that an entity appears in the users list of a
// ceph_cluster_users data source.
func testAccCheckUserInList(addr, entity string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		for k, v := range rs.Primary.Attributes {
			if strings.HasSuffix(k, ".entity") && v == entity {
				return nil
			}
		}
		return fmt.Errorf("entity %q not found in %s users list", entity, addr)
	}
}

const testAccClusterUsersDataSourceConfig = `
resource "ceph_cluster_user" "test" {
  entity       = "client.tf-acc-list"
  capabilities = { mon = "allow r" }
}

data "ceph_cluster_users" "all" {
  depends_on = [ceph_cluster_user.test]
}
`
