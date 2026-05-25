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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClusterCapacityDataSource(t *testing.T) {
	const addr = "data.ceph_cluster_capacity.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterCapacityDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(addr, "total_bytes", positiveInt64),
					resource.TestCheckResourceAttrWith(addr, "total_avail_bytes", nonNegativeInt64),
					resource.TestCheckResourceAttrWith(addr, "total_used_raw_bytes", nonNegativeInt64),
					testAccCheckCapacityConsistent(addr),
				),
			},
		},
	})
}

func positiveInt64(value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("not an integer: %q: %w", value, err)
	}
	if n <= 0 {
		return fmt.Errorf("expected a positive value, got %d", n)
	}
	return nil
}

func nonNegativeInt64(value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("not an integer: %q: %w", value, err)
	}
	if n < 0 {
		return fmt.Errorf("expected a non-negative value, got %d", n)
	}
	return nil
}

// testAccCheckCapacityConsistent verifies that used + available equals the total
// raw capacity, which Ceph reports as exact byte counts from the same source.
func testAccCheckCapacityConsistent(addr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		attrs := rs.Primary.Attributes

		total, err := strconv.ParseInt(attrs["total_bytes"], 10, 64)
		if err != nil {
			return fmt.Errorf("total_bytes: %w", err)
		}
		avail, err := strconv.ParseInt(attrs["total_avail_bytes"], 10, 64)
		if err != nil {
			return fmt.Errorf("total_avail_bytes: %w", err)
		}
		used, err := strconv.ParseInt(attrs["total_used_raw_bytes"], 10, 64)
		if err != nil {
			return fmt.Errorf("total_used_raw_bytes: %w", err)
		}

		if used+avail != total {
			return fmt.Errorf("expected total_used_raw_bytes(%d) + total_avail_bytes(%d) == total_bytes(%d), got %d",
				used, avail, total, used+avail)
		}
		return nil
	}
}

const testAccClusterCapacityDataSourceConfig = `
data "ceph_cluster_capacity" "test" {}
`
