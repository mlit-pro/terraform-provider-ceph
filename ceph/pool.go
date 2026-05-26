/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Pool is a Ceph pool as returned by GET /api/pool. Note that the read field
// `pool` is the numeric id, while the create request reuses `pool` for the name.
type Pool struct {
	Pool                int64    `json:"pool"`
	PoolName            string   `json:"pool_name"`
	Type                string   `json:"type"`
	Size                int64    `json:"size"`
	MinSize             int64    `json:"min_size"`
	PgNum               int64    `json:"pg_num"`
	CrushRule           string   `json:"crush_rule"`
	ApplicationMetadata []string `json:"application_metadata"`
	PgAutoscaleMode     string   `json:"pg_autoscale_mode"`
	QuotaMaxBytes       int64    `json:"quota_max_bytes"`
	QuotaMaxObjects     int64    `json:"quota_max_objects"`
	ErasureCodeProfile  string   `json:"erasure_code_profile"`
}

// PoolSpec is the set of create/update inputs. Pointer fields are optional: nil
// means "leave unset / do not change". Name is the current name (request body
// `pool` on create, URL path on update); Rename carries a new name on update.
type PoolSpec struct {
	Name                string
	PoolType            string
	PgNum               *int64
	Size                *int64
	MinSize             *int64
	QuotaMaxBytes       *int64
	QuotaMaxObjects     *int64
	ErasureCodeProfile  *string
	CrushRule           *string
	PgAutoscaleMode     *string
	Rename              *string
	ApplicationMetadata *[]string
}

// GetPools returns all pools from GET /api/pool.
func (c *Client) GetPools(ctx context.Context) ([]Pool, error) {
	var out []Pool
	_, err := c.Get(ctx, "/api/pool", &out)
	return out, err
}

// GetPool returns a single pool by name. It returns ErrNotFound when no pool
// with that name exists.
func (c *Client) GetPool(ctx context.Context, name string) (Pool, error) {
	pools, err := c.GetPools(ctx)
	if err != nil {
		return Pool{}, err
	}
	for _, p := range pools {
		if p.PoolName == name {
			return p, nil
		}
	}
	return Pool{}, ErrNotFound
}

// CreatePool creates a pool via POST /api/pool, waiting for the async task.
// omitempty drops unset (nil) fields; a non-nil pointer (e.g. &0) is still sent.
func (c *Client) CreatePool(ctx context.Context, s PoolSpec) error {
	body := struct {
		Pool                string    `json:"pool"`
		PoolType            string    `json:"pool_type"`
		PgNum               *int64    `json:"pg_num,omitempty"`
		ErasureCodeProfile  *string   `json:"erasure_code_profile,omitempty"`
		RuleName            *string   `json:"rule_name,omitempty"`
		ApplicationMetadata *[]string `json:"application_metadata,omitempty"`
		Size                *int64    `json:"size,omitempty"`
		MinSize             *int64    `json:"min_size,omitempty"`
		PgAutoscaleMode     *string   `json:"pg_autoscale_mode,omitempty"`
		QuotaMaxBytes       *int64    `json:"quota_max_bytes,omitempty"`
		QuotaMaxObjects     *int64    `json:"quota_max_objects,omitempty"`
	}{
		Pool:                s.Name,
		PoolType:            s.PoolType,
		PgNum:               s.PgNum,
		ErasureCodeProfile:  s.ErasureCodeProfile,
		RuleName:            s.CrushRule,
		ApplicationMetadata: s.ApplicationMetadata,
		Size:                s.Size,
		MinSize:             s.MinSize,
		PgAutoscaleMode:     s.PgAutoscaleMode,
		QuotaMaxBytes:       s.QuotaMaxBytes,
		QuotaMaxObjects:     s.QuotaMaxObjects,
	}
	status, err := c.Post(ctx, "/api/pool", body, nil)
	if err := c.awaitPoolTask(ctx, status, err, "create", s.Name); err != nil {
		return err
	}
	return c.waitForPoolConsistency(ctx, s.Name, s)
}

// UpdatePool updates a pool via PUT /api/pool/{name}, waiting for the async task.
// Only the set spec fields are sent. A non-nil Rename renames the pool. The crush
// rule uses `crush_rule` here vs `rule_name` on create.
func (c *Client) UpdatePool(ctx context.Context, s PoolSpec) error {
	body := struct {
		Pool                *string   `json:"pool,omitempty"`
		CrushRule           *string   `json:"crush_rule,omitempty"`
		PgNum               *int64    `json:"pg_num,omitempty"`
		ApplicationMetadata *[]string `json:"application_metadata,omitempty"`
		Size                *int64    `json:"size,omitempty"`
		MinSize             *int64    `json:"min_size,omitempty"`
		PgAutoscaleMode     *string   `json:"pg_autoscale_mode,omitempty"`
		QuotaMaxBytes       *int64    `json:"quota_max_bytes,omitempty"`
		QuotaMaxObjects     *int64    `json:"quota_max_objects,omitempty"`
	}{
		Pool:                s.Rename,
		CrushRule:           s.CrushRule,
		PgNum:               s.PgNum,
		ApplicationMetadata: s.ApplicationMetadata,
		Size:                s.Size,
		MinSize:             s.MinSize,
		PgAutoscaleMode:     s.PgAutoscaleMode,
		QuotaMaxBytes:       s.QuotaMaxBytes,
		QuotaMaxObjects:     s.QuotaMaxObjects,
	}
	path := "/api/pool/" + url.PathEscape(s.Name)
	status, err := c.Put(ctx, path, body, nil)
	if err := c.awaitPoolTask(ctx, status, err, "edit", s.Name); err != nil {
		return err
	}
	name := s.Name
	if s.Rename != nil {
		name = *s.Rename
	}
	return c.waitForPoolConsistency(ctx, name, s)
}

// DeletePool deletes a pool via DELETE /api/pool/{name}, waiting for the task.
func (c *Client) DeletePool(ctx context.Context, name string) error {
	path := "/api/pool/" + url.PathEscape(name)
	status, err := c.Delete(ctx, path, nil)
	return c.awaitPoolTask(ctx, status, err, "delete", name)
}

// awaitPoolTask handles a pool write's response: the verb already validated the
// status (any 2xx), so on a 202 the operation runs asynchronously and we poll
// the matching task to completion.
func (c *Client) awaitPoolTask(ctx context.Context, status int, err error, action, poolName string) error {
	if err != nil {
		return err
	}
	if status == http.StatusAccepted {
		return c.waitForTask(ctx, "pool/"+action, map[string]string{"pool_name": poolName})
	}
	return nil
}

// waitForPoolConsistency polls until the named pool reflects the fields set in s,
// or until a timeout. The mgr's cached osd_map can briefly lag a write (the value
// applied via `osd pool set` is not immediately visible on read), which would
// otherwise surface to Terraform as an "inconsistent result after apply".
func (c *Client) waitForPoolConsistency(ctx context.Context, name string, s PoolSpec) error {
	const timeout = 30 * time.Second
	deadline := time.Now().Add(timeout)
	for {
		pool, err := c.GetPool(ctx, name)
		switch {
		case err == nil && poolMatchesSpec(pool, s):
			return nil
		case err != nil && !errors.Is(err, ErrNotFound):
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("pool %q did not reflect the requested configuration within %s", name, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// poolMatchesSpec reports whether a read pool reflects the fields set in s. pg_num
// is only checked when autoscale is explicitly off, since the autoscaler manages
// it otherwise.
func poolMatchesSpec(p Pool, s PoolSpec) bool {
	if s.PgAutoscaleMode != nil && p.PgAutoscaleMode != *s.PgAutoscaleMode {
		return false
	}
	if s.Size != nil && p.Size != *s.Size {
		return false
	}
	if s.MinSize != nil && p.MinSize != *s.MinSize {
		return false
	}
	if s.QuotaMaxBytes != nil && p.QuotaMaxBytes != *s.QuotaMaxBytes {
		return false
	}
	if s.QuotaMaxObjects != nil && p.QuotaMaxObjects != *s.QuotaMaxObjects {
		return false
	}
	if s.CrushRule != nil && p.CrushRule != *s.CrushRule {
		return false
	}
	if s.PgNum != nil && s.PgAutoscaleMode != nil && *s.PgAutoscaleMode == "off" && p.PgNum != *s.PgNum {
		return false
	}
	if s.ApplicationMetadata != nil && !sameStringSet(p.ApplicationMetadata, *s.ApplicationMetadata) {
		return false
	}
	return true
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, x := range a {
		seen[x] = struct{}{}
	}
	for _, x := range b {
		if _, ok := seen[x]; !ok {
			return false
		}
	}
	return true
}
