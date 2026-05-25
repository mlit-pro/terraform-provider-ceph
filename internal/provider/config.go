/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"net/url"

	"github.com/mlit-pro/terraform-provider-ceph/internal/utils"
)

// config holds provider settings after merging HCL with environment variables.
type config struct {
	endpoint string
	username string
	password string
	caCert   string
	insecure bool
}

// newConfig builds a config from the provider model, falling back to CEPH_* environment
// variables for any attribute that is not set in the configuration.
func newConfig(data CephProviderModel) config {
	return config{
		endpoint: utils.StringOrEnv(data.Endpoint, "CEPH_ENDPOINT"),
		username: utils.StringOrEnv(data.Username, "CEPH_USERNAME"),
		password: utils.StringOrEnv(data.Password, "CEPH_PASSWORD"),
		caCert:   utils.StringOrEnv(data.CACert, "CEPH_CA_CERT"),
		insecure: utils.BoolOrEnv(data.Insecure, "CEPH_INSECURE"),
	}
}

// validationError describes a single configuration problem and the attribute it belongs to.
type validationError struct {
	attribute string
	summary   string
	detail    string
}

// validate checks required fields, endpoint URL shape, and TLS option conflicts.
func (c config) validate() []validationError {
	var errs []validationError

	if c.endpoint == "" {
		errs = append(errs, validationError{"endpoint", "Missing Ceph endpoint",
			"Set the `endpoint` attribute or the CEPH_ENDPOINT environment variable."})
	} else if u, err := url.Parse(c.endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, validationError{"endpoint", "Invalid Ceph endpoint",
			"The endpoint must be an absolute http or https URL, e.g. https://ceph-mgr.example.local:8443."})
	}

	if c.username == "" {
		errs = append(errs, validationError{"username", "Missing Ceph username",
			"Set the `username` attribute or the CEPH_USERNAME environment variable."})
	}

	if c.password == "" {
		errs = append(errs, validationError{"password", "Missing Ceph password",
			"Set the `password` attribute or the CEPH_PASSWORD environment variable."})
	}

	if c.insecure && c.caCert != "" {
		errs = append(errs, validationError{"insecure", "Conflicting TLS settings",
			"`insecure` and `ca_cert` are mutually exclusive; set at most one."})
	}

	return errs
}
