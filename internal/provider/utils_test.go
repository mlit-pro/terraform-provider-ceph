/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"fmt"
	"strconv"
)

// atLeastOne validates that a Terraform "#"/"%" count attribute is a positive
// integer. Use with resource.TestCheckResourceAttrWith.
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
