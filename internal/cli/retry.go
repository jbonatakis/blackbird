package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runRetry(taskID string) error {
	if taskID == "" {
		return UsageError{Message: "retry requires exactly 1 argument: <taskID>"}
	}

	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}
	it, ok := g.Items[taskID]
	if !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}
	if it.Status != plan.StatusFailed && it.Status != plan.StatusBlocked {
		return fmt.Errorf("task %s is not failed or blocked", taskID)
	}

	baseDir := filepath.Dir(path)
	runs, err := execution.ListRuns(baseDir, taskID)
	if err != nil {
		return err
	}
	failedRun := false
	for _, run := range runs {
		if run.Status == execution.RunStatusFailed {
			failedRun = true
			break
		}
	}
	if !failedRun {
		return fmt.Errorf("no failed runs found for %s", taskID)
	}

	if err := execution.UpdateTaskStatus(path, taskID, plan.StatusTodo); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "reset %s to todo\n", taskID)
	return nil
}
