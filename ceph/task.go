/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ceph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// cephTask is one entry from GET /api/task.
type cephTask struct {
	Name      string          `json:"name"`
	Metadata  map[string]any  `json:"metadata"`
	Exception json.RawMessage `json:"exception"`
}

// matches reports whether the task has the given name and every metadata pair.
func (t cephTask) matches(name string, metadata map[string]string) bool {
	if t.Name != name {
		return false
	}
	for k, v := range metadata {
		if fmt.Sprint(t.Metadata[k]) != v {
			return false
		}
	}
	return true
}

func (t cephTask) failed() bool {
	s := strings.TrimSpace(string(t.Exception))
	return s != "" && s != "null"
}

func (t cephTask) errorDetail() string {
	var e struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(t.Exception, &e) == nil && e.Detail != "" {
		return e.Detail
	}
	return strings.TrimSpace(string(t.Exception))
}

// waitForTask polls GET /api/task until the task identified by name and the
// given metadata finishes. It returns nil on success and an error carrying the
// task's exception detail on failure. A task that was executing but is then
// absent from both lists is treated as finished, because the dashboard prunes
// finished tasks (keeps the 10 most recent, 60s TTL). Polling stops when ctx is
// cancelled (e.g. the resource operation timeout).
func (c *Client) waitForTask(ctx context.Context, name string, metadata map[string]string) error {
	for {
		var tasks struct {
			ExecutingTasks []cephTask `json:"executing_tasks"`
			FinishedTasks  []cephTask `json:"finished_tasks"`
		}
		if _, err := c.Get(ctx, "/api/task", &tasks); err != nil {
			return err
		}

		for _, t := range tasks.FinishedTasks {
			if t.matches(name, metadata) {
				if t.failed() {
					return fmt.Errorf("task %s failed: %s", name, t.errorDetail())
				}
				return nil
			}
		}

		executing := false
		for _, t := range tasks.ExecutingTasks {
			if t.matches(name, metadata) {
				executing = true
				break
			}
		}
		if !executing {
			// Finished and already pruned (or completed between polls): treat as done.
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
