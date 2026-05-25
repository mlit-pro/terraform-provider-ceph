/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNewConfigFromEnv(t *testing.T) {
	t.Setenv("CEPH_ENDPOINT", "https://env:8443")
	t.Setenv("CEPH_USERNAME", "env-user")
	t.Setenv("CEPH_PASSWORD", "env-pass")
	t.Setenv("CEPH_CA_CERT", "env-pem")
	t.Setenv("CEPH_INSECURE", "true")

	got := newConfig(CephProviderModel{})
	want := config{
		endpoint: "https://env:8443",
		username: "env-user",
		password: "env-pass",
		caCert:   "env-pem",
		insecure: true,
	}
	if got != want {
		t.Errorf("newConfig from env = %+v, want %+v", got, want)
	}
}

func TestNewConfigPrefersConfigOverEnv(t *testing.T) {
	t.Setenv("CEPH_ENDPOINT", "https://env:8443")
	t.Setenv("CEPH_INSECURE", "true")

	got := newConfig(CephProviderModel{
		Endpoint: types.StringValue("https://config:8443"),
		Insecure: types.BoolValue(false),
	})
	if got.endpoint != "https://config:8443" {
		t.Errorf("endpoint = %q, want config value to win", got.endpoint)
	}
	if got.insecure {
		t.Error("insecure = true, want config value (false) to win over truthy env")
	}
	// Fields absent from both config and env stay empty.
	if got.username != "" {
		t.Errorf("username = %q, want empty (no config, no env)", got.username)
	}
}

func TestConfigValidate(t *testing.T) {
	valid := config{endpoint: "https://ceph:8443", username: "u", password: "p"}
	if errs := valid.validate(); len(errs) != 0 {
		t.Errorf("valid config produced errors: %+v", errs)
	}

	tests := map[string]struct {
		cfg      config
		wantAttr string
	}{
		"missing endpoint": {config{username: "u", password: "p"}, "endpoint"},
		"missing username": {config{endpoint: "https://ceph:8443", password: "p"}, "username"},
		"missing password": {config{endpoint: "https://ceph:8443", username: "u"}, "password"},
		"bad url":          {config{endpoint: "ceph:8443", username: "u", password: "p"}, "endpoint"},
		"tls conflict": {config{
			endpoint: "https://ceph:8443", username: "u", password: "p",
			caCert: "PEM", insecure: true,
		}, "insecure"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			errs := tc.cfg.validate()
			if len(errs) == 0 {
				t.Fatalf("expected an error mentioning %q", tc.wantAttr)
			}
			found := false
			for _, e := range errs {
				if e.attribute == tc.wantAttr {
					found = true
				}
			}
			if !found {
				t.Errorf("no error for attribute %q; got %+v", tc.wantAttr, errs)
			}
		})
	}
}
