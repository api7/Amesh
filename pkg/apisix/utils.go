package apisix

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
			_ = rn
			// TODO
			//if (ro.Revision !=  rn.Revision) {
			//	updated = append(updated, rn)
			//}
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
			_ = un
			// TODO
			//if (uo.Revision != un.Revisioner) {
			//	updated = append(updated, un)
			//}
		}
	}
	return
}
