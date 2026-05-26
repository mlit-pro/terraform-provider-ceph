/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// ClusterUser is a CephX cluster user (a `ceph auth` entity) as returned by
// GET /api/cluster/user. The secret key is masked in this response and is only
// obtainable via ExportClusterUsers.
type ClusterUser struct {
	Entity string            `json:"entity"`
	Caps   map[string]string `json:"caps"`
}

// userCapability is one {entity, cap} pair in a create/edit request body.
type userCapability struct {
	Entity string `json:"entity"`
	Cap    string `json:"cap"`
}

func capabilitiesFromMap(caps map[string]string) []userCapability {
	out := make([]userCapability, 0, len(caps))
	for entity, cap := range caps {
		out = append(out, userCapability{Entity: entity, Cap: cap})
	}
	return out
}

// GetClusterUsers returns all CephX cluster users from GET /api/cluster/user.
func (c *Client) GetClusterUsers(ctx context.Context) ([]ClusterUser, error) {
	var out []ClusterUser
	err := c.getJSON(ctx, "/api/cluster/user", "v1.0", &out)
	return out, err
}

// GetClusterUser returns a single cluster user by entity. The API has no
// single-user endpoint, so it lists all users and filters. The bool reports
// whether the entity was found.
func (c *Client) GetClusterUser(ctx context.Context, entity string) (ClusterUser, bool, error) {
	users, err := c.GetClusterUsers(ctx)
	if err != nil {
		return ClusterUser{}, false, err
	}
	for _, u := range users {
		if u.Entity == entity {
			return u, true, nil
		}
	}
	return ClusterUser{}, false, nil
}

// CreateClusterUser creates a CephX user via POST /api/cluster/user.
func (c *Client) CreateClusterUser(ctx context.Context, entity string, caps map[string]string) error {
	body := map[string]any{
		"user_entity":  entity,
		"capabilities": capabilitiesFromMap(caps),
	}
	_, err := c.doPlainText(ctx, http.MethodPost, "/api/cluster/user", "v1.0", body, http.StatusCreated)
	return err
}

// UpdateClusterUser overwrites a CephX user's capabilities via
// PUT /api/cluster/user.
func (c *Client) UpdateClusterUser(ctx context.Context, entity string, caps map[string]string) error {
	body := map[string]any{
		"user_entity":  entity,
		"capabilities": capabilitiesFromMap(caps),
	}
	_, err := c.doPlainText(ctx, http.MethodPut, "/api/cluster/user", "v1.0", body, http.StatusOK)
	return err
}

// DeleteClusterUser deletes a CephX user via DELETE /api/cluster/user/{entity}.
func (c *Client) DeleteClusterUser(ctx context.Context, entity string) error {
	path := "/api/cluster/user/" + url.PathEscape(entity)
	_, err := c.doPlainText(ctx, http.MethodDelete, path, "v1.0", nil, http.StatusNoContent, http.StatusOK)
	return err
}

// ExportClusterUsers returns the keyring text (including secret keys) for the
// given entities via POST /api/cluster/user/export.
func (c *Client) ExportClusterUsers(ctx context.Context, entities []string) (string, error) {
	body := map[string]any{"entities": entities}
	payload, err := c.doPlainText(ctx, http.MethodPost, "/api/cluster/user/export", "v1.0", body, http.StatusOK)
	if err != nil {
		return "", err
	}
	// The export endpoint returns a JSON string literal containing the keyring.
	var keyring string
	if json.Unmarshal(payload, &keyring) == nil {
		return keyring, nil
	}
	return string(payload), nil
}

// parseKeyringSecret returns the value of the first `key = ...` line in a
// keyring, or "" if none is present.
func parseKeyringSecret(keyring string) string {
	for line := range strings.SplitSeq(keyring, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "key"); ok {
			if _, value, found := strings.Cut(rest, "="); found {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// GetClusterUserKeyring exports a single entity and returns the full keyring
// text and the parsed secret key.
func (c *Client) GetClusterUserKeyring(ctx context.Context, entity string) (keyring, key string, err error) {
	keyring, err = c.ExportClusterUsers(ctx, []string{entity})
	if err != nil {
		return "", "", err
	}
	return keyring, parseKeyringSecret(keyring), nil
}
