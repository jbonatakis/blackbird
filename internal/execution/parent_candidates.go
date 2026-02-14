package execution

import "github.com/jbonatakis/blackbird/internal/plan"

// ParentReviewCandidateIDs returns ancestor parent task IDs that are eligible
// for parent review after changedChildID transitions to done.
//
// A parent is eligible when it is a container task (non-empty child IDs) and
// every listed child exists and is in done status.
//
// Output order is deterministic: nearest parent first, then each ancestor.
func ParentReviewCandidateIDs(g plan.WorkGraph, changedChildID string) []string {
	if changedChildID == "" {
		return []string{}
	}

	candidates := []string{}
	seen := map[string]struct{}{}
	currentID := changedChildID

	for {
		current, ok := g.Items[currentID]
		if !ok || current.ParentID == nil || *current.ParentID == "" {
			break
		}

		parentID := *current.ParentID
		if _, duplicate := seen[parentID]; duplicate {
			break
		}
		seen[parentID] = struct{}{}

		parent, ok := g.Items[parentID]
		if !ok {
			break
		}
		if isParentReviewCandidate(g, parent) {
			candidates = append(candidates, parentID)
		}

		currentID = parentID
	}

	return candidates
}

func isParentReviewCandidate(g plan.WorkGraph, parent plan.WorkItem) bool {
	if len(parent.ChildIDs) == 0 {
		return false
	}
	for _, childID := range parent.ChildIDs {
		child, ok := g.Items[childID]
		if !ok || child.Status != plan.StatusDone {
			return false
		}
	}
	return true
}
