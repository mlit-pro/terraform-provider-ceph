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

func TestAccRoleDataSource(t *testing.T) {
	const addr = "data.ceph_role.administrator"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "ceph_role" "administrator" { name = "administrator" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "administrator"),
					resource.TestCheckResourceAttr(addr, "system", "true"),
					resource.TestCheckResourceAttrWith(addr, "scopes_permissions.%", atLeastOne),
				),
			},
		},
	})
}
