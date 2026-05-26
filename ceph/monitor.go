/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import "context"

// Monitor is a single Ceph monitor from the monmap returned by GET /api/monitor.
type Monitor struct {
	Name       string `json:"name"`
	Addr       string `json:"addr"`
	Priority   int64  `json:"priority"`
	PublicAddr string `json:"public_addr"`
	Rank       int64  `json:"rank"`
	Weight     int64  `json:"weight"`
	// InQuorum reports whether the monitor is currently part of the quorum. It
	// is derived from mon_status.quorum (a list of ranks), not a monmap field.
	InQuorum bool `json:"-"`
}

// GetMonitors returns the cluster monitors from GET /api/monitor. The monmap
// (full membership) is nested under mon_status; each monitor's InQuorum is set
// from the mon_status.quorum rank list (the subset currently in quorum).
func (c *Client) GetMonitors(ctx context.Context) ([]Monitor, error) {
	var raw struct {
		MonStatus struct {
			Quorum []int64 `json:"quorum"`
			MonMap struct {
				Mons []Monitor `json:"mons"`
			} `json:"monmap"`
		} `json:"mon_status"`
	}
	if _, err := c.Get(ctx, "/api/monitor", &raw); err != nil {
		return nil, err
	}

	inQuorum := make(map[int64]bool, len(raw.MonStatus.Quorum))
	for _, rank := range raw.MonStatus.Quorum {
		inQuorum[rank] = true
	}

	mons := raw.MonStatus.MonMap.Mons
	for i := range mons {
		mons[i].InQuorum = inQuorum[mons[i].Rank]
	}
	return mons, nil
}
