/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewInvalidCACert(t *testing.T) {
	_, err := New("https://ceph:8443", "u", "p", "not a pem", false)
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}

func TestNewInsecureSetsSkipVerify(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tr, ok := c.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.http.Transport)
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewTrimsTrailingSlash(t *testing.T) {
	c, err := New("https://ceph:8443/", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.endpoint != "https://ceph:8443" {
		t.Errorf("endpoint = %q, want %q", c.endpoint, "https://ceph:8443")
	}
}

// caCertPEM extracts the test server's certificate as PEM so the client can trust it.
func caCertPEM(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	cert := srv.Certificate()
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

func TestAuthenticateHappyPath(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/auth" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.ceph.api.v1.0+json" {
			t.Errorf("Accept = %q", got)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"abc123"}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", caCertPEM(t, srv), false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if c.token != "abc123" {
		t.Errorf("token = %q, want %q", c.token, "abc123")
	}
}

func TestAuthenticateUnauthorized(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"invalid credentials"}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", caCertPEM(t, srv), false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = c.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not mention status 401", err.Error())
	}
}

func TestDoSetsAuthAndVersionHeaders(t *testing.T) {
	var gotAuth, gotAccept string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", caCertPEM(t, srv), false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.token = "tok"

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/pool", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.Do(req, "v1.1")
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotAccept != "application/vnd.ceph.api.v1.1+json" {
		t.Errorf("Accept = %q, want %q", gotAccept, "application/vnd.ceph.api.v1.1+json")
	}
}
