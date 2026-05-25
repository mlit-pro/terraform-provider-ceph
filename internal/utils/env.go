/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package utils

import (
	"os"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringOrEnv returns the attribute value if set, otherwise the environment variable.
func StringOrEnv(v types.String, env string) string {
	if !v.IsNull() {
		return v.ValueString()
	}
	return os.Getenv(env)
}

// BoolOrEnv returns the attribute value if set, otherwise true when the environment
// variable is a truthy value ("1", "true", "TRUE", etc.).
func BoolOrEnv(v types.Bool, env string) bool {
	if !v.IsNull() {
		return v.ValueBool()
	}
	switch os.Getenv(env) {
	case "1", "t", "T", "true", "TRUE", "True", "yes", "YES":
		return true
	default:
		return false
	}
}
