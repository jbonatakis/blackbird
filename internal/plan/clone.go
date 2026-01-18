package plan

func Clone(g WorkGraph) WorkGraph {
	out := WorkGraph{
		SchemaVersion: g.SchemaVersion,
		Items:         map[string]WorkItem{},
	}
	if g.Items == nil {
		out.Items = nil
		return out
	}

	for id, it := range g.Items {
		out.Items[id] = cloneItem(it)
	}
	return out
}

func cloneItem(it WorkItem) WorkItem {
	out := it
	if it.ParentID != nil {
		parent := *it.ParentID
		out.ParentID = &parent
	}
	if it.AcceptanceCriteria != nil {
		out.AcceptanceCriteria = append([]string{}, it.AcceptanceCriteria...)
	}
	if it.ChildIDs != nil {
		out.ChildIDs = append([]string{}, it.ChildIDs...)
	}
	if it.Deps != nil {
		out.Deps = append([]string{}, it.Deps...)
	}
	if it.Notes != nil {
		n := *it.Notes
		out.Notes = &n
	}
	if it.DepRationale != nil {
		out.DepRationale = copyRationale(it.DepRationale)
	}
	return out
}
