/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

// Package ceph is a minimal REST client for the Ceph Manager Dashboard API.
// It owns TLS configuration and JWT authentication. Requests go through the
// Get/Post/Put/Delete helpers; every endpoint in use is API v1.0 (see apiVersion).
package ceph

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// apiVersion is the Ceph Dashboard API version negotiated via the Accept header.
const apiVersion = "v1.0"

// ErrNotFound is returned by the Get* lookups when the requested resource does
// not exist. Callers can test for it with errors.Is.
var ErrNotFound = errors.New("ceph: resource not found")

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
	req.Header.Set("Accept", "application/vnd.ceph.api."+apiVersion+"+json")

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

// RequestOption configures a request issued by Get/Post/Put/Delete.
type RequestOption func(*requestOptions)

type requestOptions struct {
	apiVersion string
	okStatuses []int
}

// WithAPIVersion overrides the API version (default "v1.0") used for a request's
// Accept header, supporting per-endpoint versioning.
func WithAPIVersion(version string) RequestOption {
	return func(o *requestOptions) { o.apiVersion = version }
}

// WithStatuses restricts the response status codes treated as success. By
// default any 2xx is accepted.
func WithStatuses(statuses ...int) RequestOption {
	return func(o *requestOptions) { o.okStatuses = statuses }
}

// Get issues a GET, decoding a JSON response body into out when out is non-nil.
func (c *Client) Get(ctx context.Context, path string, out any, opts ...RequestOption) (int, error) {
	return c.do(ctx, http.MethodGet, path, nil, out, opts)
}

// Post issues a POST with a JSON body, decoding a JSON response into out when non-nil.
func (c *Client) Post(ctx context.Context, path string, body, out any, opts ...RequestOption) (int, error) {
	return c.do(ctx, http.MethodPost, path, body, out, opts)
}

// Put issues a PUT with a JSON body, decoding a JSON response into out when non-nil.
func (c *Client) Put(ctx context.Context, path string, body, out any, opts ...RequestOption) (int, error) {
	return c.do(ctx, http.MethodPut, path, body, out, opts)
}

// Delete issues a DELETE, decoding a JSON response into out when non-nil.
func (c *Client) Delete(ctx context.Context, path string, out any, opts ...RequestOption) (int, error) {
	return c.do(ctx, http.MethodDelete, path, nil, out, opts)
}

// do builds, sends, and reads a request. The API version defaults to apiVersion
// and the body is JSON-encoded when non-nil. The status must be in the request's
// okStatuses, or any 2xx when none are given; otherwise an error is returned.
// When out is non-nil and the status is acceptable, the JSON body is decoded into
// it. The status code is always returned (e.g. to distinguish 202).
func (c *Client) do(ctx context.Context, method, path string, body, out any, opts []RequestOption) (int, error) {
	cfg := requestOptions{apiVersion: apiVersion}
	for _, opt := range opts {
		opt(&cfg)
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("encoding request for %s: %w", path, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reader)
	if err != nil {
		return 0, fmt.Errorf("building request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.ceph.api."+cfg.apiVersion+"+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("sending request to %s%s: %w", c.endpoint, path, err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(resp.Body)

	if !statusAccepted(resp.StatusCode, cfg.okStatuses) {
		return resp.StatusCode, fmt.Errorf("request to %s failed: %d: %s", path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decoding response from %s: %w", path, err)
		}
	}

	return resp.StatusCode, nil
}

// statusAccepted reports whether status is allowed: a member of okStatuses, or
// any 2xx when okStatuses is empty.
func statusAccepted(status int, okStatuses []int) bool {
	if len(okStatuses) == 0 {
		return status >= 200 && status < 300
	}
	return slices.Contains(okStatuses, status)
}
