package agent

import (
	"fmt"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func ApplyPatch(g *plan.WorkGraph, ops []PatchOp, now time.Time) error {
	if g == nil {
		return fmt.Errorf("plan graph is nil")
	}
	if g.Items == nil {
		g.Items = map[string]plan.WorkItem{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	for i, op := range ops {
		switch op.Op {
		case PatchAdd:
			if op.Item == nil {
				return fmt.Errorf("patch[%d] add: item is required", i)
			}
			parentID := op.ParentID
			if parentID == nil && op.Item.ParentID != nil && *op.Item.ParentID != "" {
				pid := *op.Item.ParentID
				parentID = &pid
			}
			if err := plan.AddItem(g, copyWorkItem(*op.Item), parentID, op.Index, now); err != nil {
				return fmt.Errorf("patch[%d] add: %w", i, err)
			}
		case PatchUpdate:
			if op.Item == nil {
				return fmt.Errorf("patch[%d] update: item is required", i)
			}
			id := op.ID
			if id == "" {
				id = op.Item.ID
			}
			if id == "" {
				return fmt.Errorf("patch[%d] update: id is required", i)
			}
			existing, ok := g.Items[id]
			if !ok {
				return fmt.Errorf("patch[%d] update: unknown id %q", i, id)
			}
			updated := existing
			updated.Title = op.Item.Title
			updated.Description = op.Item.Description
			updated.AcceptanceCriteria = append([]string{}, op.Item.AcceptanceCriteria...)
			updated.Prompt = op.Item.Prompt
			updated.Status = op.Item.Status
			updated.Deps = append([]string{}, op.Item.Deps...)
			updated.DepRationale = copyRationale(op.Item.DepRationale)
			if op.Item.Notes != nil {
				n := *op.Item.Notes
				updated.Notes = &n
			} else {
				updated.Notes = nil
			}
			updated.UpdatedAt = now
			g.Items[id] = updated
		case PatchDelete:
			if op.ID == "" {
				return fmt.Errorf("patch[%d] delete: id is required", i)
			}
			if _, err := plan.DeleteItem(g, op.ID, false, false, now); err != nil {
				return fmt.Errorf("patch[%d] delete: %w", i, err)
			}
		case PatchMove:
			if op.ID == "" {
				return fmt.Errorf("patch[%d] move: id is required", i)
			}
			if err := plan.MoveItem(g, op.ID, op.ParentID, op.Index, now); err != nil {
				return fmt.Errorf("patch[%d] move: %w", i, err)
			}
		case PatchSetDeps:
			if op.ID == "" {
				return fmt.Errorf("patch[%d] set_deps: id is required", i)
			}
			if op.Deps == nil {
				return fmt.Errorf("patch[%d] set_deps: deps is required", i)
			}
			if err := plan.SetDeps(g, op.ID, op.Deps, now); err != nil {
				return fmt.Errorf("patch[%d] set_deps: %w", i, err)
			}
			if op.DepRationale != nil {
				it := g.Items[op.ID]
				it.DepRationale = copyRationale(op.DepRationale)
				it.UpdatedAt = now
				g.Items[op.ID] = it
			}
		case PatchAddDep:
			if op.ID == "" || op.DepID == "" {
				return fmt.Errorf("patch[%d] add_dep: id and depId are required", i)
			}
			if err := plan.AddDep(g, op.ID, op.DepID, now); err != nil {
				return fmt.Errorf("patch[%d] add_dep: %w", i, err)
			}
			if op.DepRationale != nil {
				it := g.Items[op.ID]
				if it.DepRationale == nil {
					it.DepRationale = map[string]string{}
				}
				if reason, ok := op.DepRationale[op.DepID]; ok {
					it.DepRationale[op.DepID] = reason
					it.UpdatedAt = now
					g.Items[op.ID] = it
				}
			}
		case PatchRemoveDep:
			if op.ID == "" || op.DepID == "" {
				return fmt.Errorf("patch[%d] remove_dep: id and depId are required", i)
			}
			if err := plan.RemoveDep(g, op.ID, op.DepID, now); err != nil {
				return fmt.Errorf("patch[%d] remove_dep: %w", i, err)
			}
		default:
			return fmt.Errorf("patch[%d]: unsupported op %q", i, op.Op)
		}
	}

	if errs := plan.Validate(*g); len(errs) != 0 {
		return fmt.Errorf("patch application produced invalid plan")
	}

	return nil
}

func copyWorkItem(it plan.WorkItem) plan.WorkItem {
	out := it
	if it.ParentID != nil {
		pid := *it.ParentID
		out.ParentID = &pid
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

func copyRationale(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
