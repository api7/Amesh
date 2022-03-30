// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package util

import (
	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/apisix"
)

// Manifest collects a couples Routes, Upstreams.
type Manifest struct {
	Routes    []*apisix.Route
	Upstreams []*apisix.Upstream
}

// DiffFrom checks the difference between m and m2 from m's point of view.
func (m *Manifest) DiffFrom(m2 *Manifest) (*Manifest, *Manifest, *Manifest) {
	var (
		added   Manifest
		updated Manifest
		deleted Manifest
	)

	a, d, u := apisix.CompareRoutes(m.Routes, m2.Routes)
	added.Routes = append(added.Routes, a...)
	updated.Routes = append(updated.Routes, u...)
	deleted.Routes = append(deleted.Routes, d...)

	au, du, uu := apisix.CompareUpstreams(m.Upstreams, m2.Upstreams)
	added.Upstreams = append(added.Upstreams, au...)
	updated.Upstreams = append(updated.Upstreams, uu...)
	deleted.Upstreams = append(deleted.Upstreams, du...)

	return &added, &deleted, &updated
}

// Size calculates the number of resources in the manifest.
func (m *Manifest) Size() int {
	return len(m.Upstreams) + len(m.Routes)
}

// Events generates events according to its collection.
func (m *Manifest) Events(evType types.EventType) []types.Event {
	var events []types.Event
	for _, r := range m.Routes {
		if evType == types.EventDelete {
			events = append(events, types.Event{
				Type:      types.EventDelete,
				Tombstone: r,
			})
		} else {
			events = append(events, types.Event{
				Type:   evType,
				Object: r,
			})
		}
	}
	for _, u := range m.Upstreams {
		if evType == types.EventDelete {
			events = append(events, types.Event{
				Type:      types.EventDelete,
				Tombstone: u,
			})
		} else {
			// TODO: We may turn update events into delete events once we handle ErrRequireFurtherEDS more properly
			if u.Nodes == nil || len(u.Nodes) == 0 {
				continue
			}
			events = append(events, types.Event{
				Type:   evType,
				Object: u,
			})
		}
	}
	return events
}
