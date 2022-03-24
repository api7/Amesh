package util

import (
	"github.com/api7/amesh/pkg/amesh/provisioner"
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
func (m *Manifest) Events(evType provisioner.EventType) []provisioner.Event {
	var events []provisioner.Event
	for _, r := range m.Routes {
		if evType == provisioner.EventDelete {
			events = append(events, provisioner.Event{
				Type:      provisioner.EventDelete,
				Tombstone: r,
			})
		} else {
			events = append(events, provisioner.Event{
				Type:   evType,
				Object: r,
			})
		}
	}
	for _, u := range m.Upstreams {
		if evType == provisioner.EventDelete {
			events = append(events, provisioner.Event{
				Type:      provisioner.EventDelete,
				Tombstone: u,
			})
		} else {
			// TODO: We may turn update events into delete events once we handle ErrRequireFurtherEDS more properly
			if u.Nodes == nil || len(u.Nodes) == 0 {
				continue
			}
			events = append(events, provisioner.Event{
				Type:   evType,
				Object: u,
			})
		}
	}
	return events
}
