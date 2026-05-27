/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import (
	"context"
	"net/http"
	"net/url"
)

// User is a Ceph Dashboard RBAC user account from GET /api/user. The password
// is never returned by the API.
type User struct {
	Username          string   `json:"username"`
	Roles             []string `json:"roles"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	Enabled           bool     `json:"enabled"`
	LastUpdate        int64    `json:"lastUpdate"`
	PwdExpirationDate *int64   `json:"pwdExpirationDate"`
	PwdUpdateRequired bool     `json:"pwdUpdateRequired"`
}

// UserSpec is the create/update payload for a dashboard user.
type UserSpec struct {
	Username          string   `json:"username"`
	Password          string   `json:"password,omitempty"`
	Roles             []string `json:"roles"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	Enabled           bool     `json:"enabled"`
	PwdExpirationDate *int64   `json:"pwdExpirationDate"`
	PwdUpdateRequired bool     `json:"pwdUpdateRequired"`
}

// GetUsers returns all dashboard users from GET /api/user.
func (c *Client) GetUsers(ctx context.Context) ([]User, error) {
	var out []User
	_, err := c.Get(ctx, "/api/user", &out)
	return out, err
}

// GetUser returns a single dashboard user from GET /api/user/{username}. It
// returns ErrNotFound when the user does not exist.
func (c *Client) GetUser(ctx context.Context, username string) (User, error) {
	var out User
	status, err := c.Get(ctx, "/api/user/"+url.PathEscape(username), &out)
	if status == http.StatusNotFound {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return out, nil
}

// CreateUser creates a dashboard user via POST /api/user.
func (c *Client) CreateUser(ctx context.Context, spec UserSpec) error {
	_, err := c.Post(ctx, "/api/user", spec, nil)
	return err
}

// UpdateUser overwrites a dashboard user via PUT /api/user/{username}.
func (c *Client) UpdateUser(ctx context.Context, spec UserSpec) error {
	_, err := c.Put(ctx, "/api/user/"+url.PathEscape(spec.Username), spec, nil)
	return err
}

// DeleteUser deletes a dashboard user via DELETE /api/user/{username}.
func (c *Client) DeleteUser(ctx context.Context, username string) error {
	_, err := c.Delete(ctx, "/api/user/"+url.PathEscape(username), nil)
	return err
}
