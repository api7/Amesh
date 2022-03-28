package apisix

import (
	"bytes"
	"encoding/json"
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
)

// CompareRoutes diffs two Route array and finds the new adds, updates
// and deleted ones. Note it stands on the first Route array's point
// of view.
func CompareRoutes(r1, r2 []*Route) (added, deleted, updated []*Route) {
	if r1 == nil {
		return r2, nil, nil
	}
	if r2 == nil {
		return nil, r1, nil
	}

	r1Map := make(map[string]*Route)
	r2Map := make(map[string]*Route)
	for _, r := range r1 {
		r1Map[r.Id] = r
	}
	for _, r := range r2 {
		r2Map[r.Id] = r
	}
	for _, r := range r2 {
		if _, ok := r1Map[r.Id]; !ok {
			added = append(added, r)
		}
	}
	for _, ro := range r1 {
		if rn, ok := r2Map[ro.Id]; !ok {
			deleted = append(deleted, ro)
		} else {
			roJson, errO := json.Marshal(ro)
			rnJson, errN := json.Marshal(rn)
			if errO != nil || errN != nil {
				log.Errorw("compare route: marshal failed",
					zap.NamedError("error_old", errO),
					zap.NamedError("error_new", errN),
				)
				updated = append(updated, rn)
			} else if bytes.Equal(roJson, rnJson) {
				updated = append(updated, rn)
			}
		}
	}
	return
}

// CompareUpstreams diffs two Upstreams array and finds the new adds, updates
// and deleted ones. Note it stands on the first Upstream array's point
// of view.
func CompareUpstreams(u1, u2 []*Upstream) (added, deleted, updated []*Upstream) {
	if u1 == nil {
		return u2, nil, nil
	}
	if u2 == nil {
		return nil, u1, nil
	}
	u1Map := make(map[string]*Upstream)
	u2Map := make(map[string]*Upstream)
	for _, u := range u1 {
		u1Map[u.Id] = u
	}
	for _, u := range u2 {
		u2Map[u.Id] = u
	}
	for _, u := range u2 {
		if _, ok := u1Map[u.Id]; !ok {
			added = append(added, u)
		}
	}
	for _, uo := range u1 {
		if un, ok := u2Map[uo.Id]; !ok {
			deleted = append(deleted, uo)
		} else {
			uoJson, errO := json.Marshal(uo)
			unJson, errN := json.Marshal(un)
			if errO != nil || errN != nil {
				log.Errorw("compare upstream: marshal failed",
					zap.NamedError("error_old", errO),
					zap.NamedError("error_new", errN),
				)
				updated = append(updated, un)
			} else if bytes.Equal(uoJson, unJson) {
				updated = append(updated, un)
			}
		}
	}
	return
}
