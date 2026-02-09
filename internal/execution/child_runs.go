package execution

import (
	"fmt"
	"strings"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// ChildRunContext captures the latest completed run context for a child task.
type ChildRunContext struct {
	ChildID string
	Child   plan.WorkItem
	Run     RunRecord
}

// GetLatestCompletedChildRuns returns each referenced child task's latest completed run
// in the parent task's ChildIDs order.
func GetLatestCompletedChildRuns(g plan.WorkGraph, baseDir, parentTaskID string) ([]ChildRunContext, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir required")
	}
	if parentTaskID == "" {
		return nil, fmt.Errorf("task id required")
	}

	parent, ok := g.Items[parentTaskID]
	if !ok {
		return nil, fmt.Errorf("unknown id %q", parentTaskID)
	}

	contexts := make([]ChildRunContext, 0, len(parent.ChildIDs))
	issues := childRunContextIssues{}

	for _, childID := range parent.ChildIDs {
		child, ok := g.Items[childID]
		if !ok {
			issues.unknownChildren = append(issues.unknownChildren, childID)
			continue
		}
		if child.Status != plan.StatusDone {
			issues.notDoneChildren = append(issues.notDoneChildren, fmt.Sprintf("%s(%s)", childID, child.Status))
			continue
		}

		latest, err := GetLatestRun(baseDir, childID)
		if err != nil {
			return nil, fmt.Errorf("latest run for child %q: %w", childID, err)
		}
		if latest == nil {
			issues.missingRuns = append(issues.missingRuns, childID)
			continue
		}
		if !isTerminalRunStatus(latest.Status) {
			issues.nonTerminalRuns = append(issues.nonTerminalRuns, fmt.Sprintf("%s(%s)", childID, latest.Status))
			continue
		}

		contexts = append(contexts, ChildRunContext{
			ChildID: childID,
			Child:   child,
			Run:     *latest,
		})
	}

	if err := issues.err(parentTaskID); err != nil {
		return nil, err
	}

	return contexts, nil
}

type childRunContextIssues struct {
	unknownChildren []string
	notDoneChildren []string
	missingRuns     []string
	nonTerminalRuns []string
}

func (i childRunContextIssues) err(parentTaskID string) error {
	if len(i.unknownChildren) == 0 &&
		len(i.notDoneChildren) == 0 &&
		len(i.missingRuns) == 0 &&
		len(i.nonTerminalRuns) == 0 {
		return nil
	}

	parts := make([]string, 0, 4)
	if len(i.unknownChildren) > 0 {
		parts = append(parts, fmt.Sprintf("unknown child ids: %s", strings.Join(i.unknownChildren, ", ")))
	}
	if len(i.notDoneChildren) > 0 {
		parts = append(parts, fmt.Sprintf("children not done: %s", strings.Join(i.notDoneChildren, ", ")))
	}
	if len(i.missingRuns) > 0 {
		parts = append(parts, fmt.Sprintf("missing completed runs: %s", strings.Join(i.missingRuns, ", ")))
	}
	if len(i.nonTerminalRuns) > 0 {
		parts = append(parts, fmt.Sprintf("latest child runs are non-terminal: %s", strings.Join(i.nonTerminalRuns, ", ")))
	}

	return fmt.Errorf("parent %q cannot build child run context: %s", parentTaskID, strings.Join(parts, "; "))
}
