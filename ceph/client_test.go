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

func TestRequestSetsAuthAndDefaultVersionHeaders(t *testing.T) {
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

	if _, err := c.Get(context.Background(), "/api/pool", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotAccept != "application/vnd.ceph.api.v1.0+json" {
		t.Errorf("Accept = %q, want %q (default version)", gotAccept, "application/vnd.ceph.api.v1.0+json")
	}
}

func TestRequestWithAPIVersionOverride(t *testing.T) {
	var gotAccept string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := New(srv.URL, "u", "p", caCertPEM(t, srv), false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.token = "tok"

	if _, err := c.Get(context.Background(), "/api/pool", nil, WithAPIVersion("v1.1")); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAccept != "application/vnd.ceph.api.v1.1+json" {
		t.Errorf("Accept = %q, want %q", gotAccept, "application/vnd.ceph.api.v1.1+json")
	}
}

func TestSetProxyHTTP(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.SetProxy("http://proxy.example.com:8080"); err != nil {
		t.Fatalf("SetProxy: %v", err)
	}

	tr, ok := c.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.http.Transport)
	}

	req, _ := http.NewRequest(http.MethodGet, "https://ceph:8443/test", nil)
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("Proxy func error: %v", err)
	}
	if proxyURL == nil {
		t.Fatal("expected proxy URL, got nil")
	}
	if proxyURL.String() != "http://proxy.example.com:8080" {
		t.Errorf("proxy URL = %q, want %q", proxyURL.String(), "http://proxy.example.com:8080")
	}
}

func TestSetProxySOCKS5(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.SetProxy("socks5://proxy.example.com:1080"); err != nil {
		t.Fatalf("SetProxy: %v", err)
	}

	tr, ok := c.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.http.Transport)
	}

	req, _ := http.NewRequest(http.MethodGet, "https://ceph:8443/test", nil)
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("Proxy func error: %v", err)
	}
	if proxyURL == nil {
		t.Fatal("expected proxy URL, got nil")
	}
	if proxyURL.String() != "socks5://proxy.example.com:1080" {
		t.Errorf("proxy URL = %q, want %q", proxyURL.String(), "socks5://proxy.example.com:1080")
	}
}

func TestSetProxyInvalidURL(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.SetProxy("://invalid")
	if err == nil {
		t.Fatal("expected error for invalid proxy URL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("error %q does not mention invalid URL", err.Error())
	}
}

func TestSetProxyEmptyString(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.SetProxy(""); err != nil {
		t.Fatalf("SetProxy with empty string: %v", err)
	}
}

func TestSetProxyHTTPS(t *testing.T) {
	c, err := New("https://ceph:8443", "u", "p", "", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.SetProxy("https://proxy.example.com:8443"); err != nil {
		t.Fatalf("SetProxy: %v", err)
	}

	tr, ok := c.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.http.Transport)
	}

	req, _ := http.NewRequest(http.MethodGet, "https://ceph:8443/test", nil)
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("Proxy func error: %v", err)
	}
	if proxyURL == nil {
		t.Fatal("expected proxy URL, got nil")
	}
	if proxyURL.String() != "https://proxy.example.com:8443" {
		t.Errorf("proxy URL = %q, want %q", proxyURL.String(), "https://proxy.example.com:8443")
	}
}
