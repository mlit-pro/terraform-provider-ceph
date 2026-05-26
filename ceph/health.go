/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import "context"

// HealthMinimal is the cluster's minimal health report from GET /api/health/minimal.
type HealthMinimal struct {
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
}

// ClusterCapacity is the raw cluster capacity from GET /api/health/get_cluster_capacity.
type ClusterCapacity struct {
	TotalAvailBytes   int64 `json:"total_avail_bytes"`
	TotalBytes        int64 `json:"total_bytes"`
	TotalUsedRawBytes int64 `json:"total_used_raw_bytes"`
}

// GetHealthMinimal returns the cluster's minimal health report from
// GET /api/health/minimal.
func (c *Client) GetHealthMinimal(ctx context.Context) (HealthMinimal, error) {
	var out HealthMinimal
	_, err := c.Get(ctx, "/api/health/minimal", &out)
	return out, err
}

// GetClusterFSID returns the cluster's FSID from GET /api/health/get_cluster_fsid.
func (c *Client) GetClusterFSID(ctx context.Context) (string, error) {
	var fsid string
	_, err := c.Get(ctx, "/api/health/get_cluster_fsid", &fsid)
	return fsid, err
}

// GetClusterCapacity returns the cluster's capacity from
// GET /api/health/get_cluster_capacity.
func (c *Client) GetClusterCapacity(ctx context.Context) (ClusterCapacity, error) {
	var out ClusterCapacity
	_, err := c.Get(ctx, "/api/health/get_cluster_capacity", &out)
	return out, err
}
