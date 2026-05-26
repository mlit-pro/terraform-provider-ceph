/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMonitorsDataSource(t *testing.T) {
	const addr = "data.ceph_monitors.all"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "ceph_monitors" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(addr, "monitors.#", atLeastOne),
					resource.TestCheckResourceAttrSet(addr, "monitors.0.name"),
					resource.TestCheckResourceAttrSet(addr, "monitors.0.public_addr"),
					resource.TestCheckResourceAttrSet(addr, "monitors.0.in_quorum"),
				),
			},
		},
	})
}

// atLeastOne validates that a "#" count attribute is a positive integer.
func atLeastOne(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("expected an integer count, got %q: %w", value, err)
	}
	if n < 1 {
		return fmt.Errorf("expected at least one element, got %d", n)
	}
	return nil
}
