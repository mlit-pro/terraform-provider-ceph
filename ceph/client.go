/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

// Package ceph is a minimal REST client for the Ceph Manager Dashboard API.
// It owns TLS configuration and JWT authentication; per-endpoint API versioning is
// the caller's responsibility (see Do).
package ceph

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// Client talks to a single Ceph Manager Dashboard endpoint.
type Client struct {
	endpoint string
	username string
	password string
	http     *http.Client
	token    string
}

// New builds a Client with TLS configured from caCertPEM and insecure. It does not
// perform any network I/O; call Authenticate to obtain a token.
func New(endpoint, username, password, caCertPEM string, insecure bool) (*Client, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	}

	if caCertPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caCertPEM)) {
			return nil, fmt.Errorf("ca_cert: no valid PEM certificates found")
		}
		tlsConfig.RootCAs = pool
	}

	return &Client{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		username: username,
		password: password,
		http: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
	}, nil
}

// Authenticate exchanges username/password for a JWT via POST /api/auth and stores it.
func (c *Client) Authenticate(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	if err != nil {
		return fmt.Errorf("encoding auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/auth", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.ceph.api.v1.0+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sending auth request to %s: %w", c.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("authentication failed: %s: %s", resp.Status, strings.TrimSpace(string(snippet)))
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}
	if out.Token == "" {
		return fmt.Errorf("authentication succeeded but no token was returned")
	}

	c.token = out.Token
	return nil
}

// Do sends req with the stored bearer token and the per-endpoint Accept header.
// apiVersion is the endpoint's required version, e.g. "v1.0" or "v1.1".
func (c *Client) Do(req *http.Request, apiVersion string) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", fmt.Sprintf("application/vnd.ceph.api.%s+json", apiVersion))
	return c.http.Do(req)
}

// getJSON performs a GET against path at the given API version and decodes a 200
// response body into out.
func (c *Client) getJSON(ctx context.Context, path, apiVersion string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}

	resp, err := c.Do(req, apiVersion)
	if err != nil {
		return fmt.Errorf("sending request to %s%s: %w", c.endpoint, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("request to %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(snippet)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response from %s: %w", path, err)
	}

	return nil
}

// doPlainText sends method to path at the given API version, optionally with a
// JSON-encoded body, and returns the raw response body. The Ceph cluster-user
// write and export endpoints reply with plain text rather than JSON. okStatuses
// lists the acceptable response status codes.
func (c *Client) doPlainText(ctx context.Context, method, path, apiVersion string, body any, okStatuses ...int) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request for %s: %w", path, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reader)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(req, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s%s: %w", c.endpoint, path, err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(resp.Body)

	if slices.Contains(okStatuses, resp.StatusCode) {
		return payload, nil
	}

	return nil, fmt.Errorf("request to %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(payload)))
}
