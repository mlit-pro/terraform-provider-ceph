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

// Role is a Ceph Dashboard RBAC role from GET /api/role. Built-in roles have
// System set to true and cannot be modified or deleted.
type Role struct {
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	ScopesPermissions map[string][]string `json:"scopes_permissions"`
	System            bool                `json:"system"`
}

// RoleSpec is the create/update payload for a dashboard role.
type RoleSpec struct {
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	ScopesPermissions map[string][]string `json:"scopes_permissions"`
}

// GetRoles returns all dashboard roles from GET /api/role.
func (c *Client) GetRoles(ctx context.Context) ([]Role, error) {
	var out []Role
	_, err := c.Get(ctx, "/api/role", &out)
	return out, err
}

// GetRole returns a single dashboard role from GET /api/role/{name}. It returns
// ErrNotFound when the role does not exist.
func (c *Client) GetRole(ctx context.Context, name string) (Role, error) {
	var out Role
	status, err := c.Get(ctx, "/api/role/"+url.PathEscape(name), &out)
	if status == http.StatusNotFound {
		return Role{}, ErrNotFound
	}
	if err != nil {
		return Role{}, err
	}
	return out, nil
}

// CreateRole creates a dashboard role via POST /api/role.
func (c *Client) CreateRole(ctx context.Context, spec RoleSpec) error {
	_, err := c.Post(ctx, "/api/role", spec, nil)
	return err
}

// UpdateRole overwrites a dashboard role via PUT /api/role/{name}.
func (c *Client) UpdateRole(ctx context.Context, spec RoleSpec) error {
	_, err := c.Put(ctx, "/api/role/"+url.PathEscape(spec.Name), spec, nil)
	return err
}

// DeleteRole deletes a dashboard role via DELETE /api/role/{name}.
func (c *Client) DeleteRole(ctx context.Context, name string) error {
	_, err := c.Delete(ctx, "/api/role/"+url.PathEscape(name), nil)
	return err
}
