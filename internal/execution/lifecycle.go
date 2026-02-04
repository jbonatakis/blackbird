package execution

import (
	"fmt"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

var allowedTransitions = map[plan.Status]map[plan.Status]bool{
	plan.StatusTodo: {
		plan.StatusQueued:     true,
		plan.StatusInProgress: true,
		plan.StatusBlocked:    true,
		plan.StatusSkipped:    true,
	},
	plan.StatusQueued: {
		plan.StatusInProgress: true,
		plan.StatusBlocked:    true,
	},
	plan.StatusInProgress: {
		plan.StatusDone:        true,
		plan.StatusFailed:      true,
		plan.StatusWaitingUser: true,
		plan.StatusBlocked:     true,
	},
	plan.StatusWaitingUser: {
		plan.StatusInProgress: true,
		plan.StatusBlocked:    true,
	},
	plan.StatusFailed: {
		plan.StatusTodo:   true,
		plan.StatusQueued: true,
	},
	plan.StatusBlocked: {
		plan.StatusTodo: true,
	},
	plan.StatusSkipped: {
		plan.StatusTodo: true,
	},
	plan.StatusDone: {
		plan.StatusInProgress: true,
		plan.StatusFailed:     true,
	},
}

// UpdateTaskStatus updates a task status with lifecycle validation and atomic persistence.
func UpdateTaskStatus(planPath string, taskID string, next plan.Status) error {
	if taskID == "" {
		return fmt.Errorf("task id required")
	}

	g, err := plan.Load(planPath)
	if err != nil {
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return fmt.Errorf("plan is invalid (run `blackbird validate`): %s", planPath)
	}

	it, ok := g.Items[taskID]
	if !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}
	if err := validateTransition(it.Status, next); err != nil {
		return err
	}

	if it.Status == next {
		return nil
	}

	it.Status = next
	it.UpdatedAt = time.Now().UTC()
	g.Items[taskID] = it

	if next == plan.StatusDone {
		plan.PropagateParentCompletion(&g, taskID, it.UpdatedAt)
	}

	if err := plan.SaveAtomic(planPath, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	return nil
}

func validateTransition(current, next plan.Status) error {
	if current == next {
		return nil
	}

	allowed, ok := allowedTransitions[current]
	if !ok {
		return fmt.Errorf("unsupported current status %q", current)
	}
	if !allowed[next] {
		return fmt.Errorf("invalid status transition: %s -> %s", current, next)
	}
	return nil
}
