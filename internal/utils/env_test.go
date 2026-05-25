/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package utils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringOrEnv(t *testing.T) {
	t.Setenv("CEPH_TEST_STRING", "from-env")

	if got := StringOrEnv(types.StringValue("from-config"), "CEPH_TEST_STRING"); got != "from-config" {
		t.Errorf("config value should win: got %q", got)
	}
	if got := StringOrEnv(types.StringNull(), "CEPH_TEST_STRING"); got != "from-env" {
		t.Errorf("null should fall back to env: got %q", got)
	}
	if got := StringOrEnv(types.StringNull(), "CEPH_TEST_UNSET"); got != "" {
		t.Errorf("unset env should be empty: got %q", got)
	}
}

func TestBoolOrEnv(t *testing.T) {
	if got := BoolOrEnv(types.BoolValue(true), "CEPH_TEST_BOOL"); !got {
		t.Error("config true should win")
	}
	if got := BoolOrEnv(types.BoolValue(false), "CEPH_TEST_BOOL"); got {
		t.Error("config false should win even with truthy env")
	}

	t.Setenv("CEPH_TEST_BOOL", "true")
	if got := BoolOrEnv(types.BoolNull(), "CEPH_TEST_BOOL"); !got {
		t.Error("null should fall back to truthy env")
	}
	t.Setenv("CEPH_TEST_BOOL", "nonsense")
	if got := BoolOrEnv(types.BoolNull(), "CEPH_TEST_BOOL"); got {
		t.Error("non-truthy env should be false")
	}
}
