/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int64Ptr returns a pointer to the attribute's value, or nil when it is null or
// unknown. Useful for building request bodies where unset fields are omitted.
func Int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	x := v.ValueInt64()
	return &x
}

// NullableString maps an empty string to a null attribute value, so optional
// string attributes round-trip with a null configuration. A non-empty string
// becomes a known value.
func NullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// StringPtr returns a pointer to the attribute's value, or nil when it is null or
// unknown.
func StringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	x := v.ValueString()
	return &x
}

// StringSetPtr converts a set attribute to a *[]string for request bodies that
// distinguish "leave unset" from "set to empty". A null or unknown set yields
// nil (the field is omitted); any present set - including an empty one - yields a
// pointer to a non-nil slice (so an explicit "clear all" is sent).
func StringSetPtr(ctx context.Context, v types.Set, diags *diag.Diagnostics) *[]string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	if out == nil {
		out = []string{}
	}
	return &out
}
