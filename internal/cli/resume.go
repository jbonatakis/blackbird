package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runResume(taskID string) error {
	if taskID == "" {
		return UsageError{Message: "resume requires exactly 1 argument: <taskID>"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}
	if _, ok := g.Items[taskID]; !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}

	baseDir := filepath.Dir(path)
	runs, err := execution.ListRuns(baseDir, taskID)
	if err != nil {
		return err
	}
	var waiting *execution.RunRecord
	for i := len(runs) - 1; i >= 0; i-- {
		if runs[i].Status == execution.RunStatusWaitingUser {
			waiting = &runs[i]
			break
		}
	}
	if waiting == nil {
		fmt.Fprintf(os.Stdout, "no waiting runs for %s\n", taskID)
		return nil
	}

	questions, err := execution.ParseQuestions(waiting.Stdout)
	if err != nil {
		return err
	}
	if len(questions) == 0 {
		return fmt.Errorf("no questions found in waiting run for %s", taskID)
	}

	answers, err := promptAnswers(questions)
	if err != nil {
		return err
	}

	ctxPack, err := execution.ResumeWithAnswer(*waiting, answers)
	if err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	if err := execution.UpdateTaskStatus(path, taskID, plan.StatusInProgress); err != nil {
		return err
	}

	record, execErr := execution.LaunchAgent(nil, runtime, ctxPack)
	if err := execution.SaveRun(baseDir, record); err != nil {
		return err
	}

	switch record.Status {
	case execution.RunStatusSuccess:
		if err := execution.UpdateTaskStatus(path, taskID, plan.StatusDone); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "completed %s\n", taskID)
	case execution.RunStatusWaitingUser:
		if err := execution.UpdateTaskStatus(path, taskID, plan.StatusWaitingUser); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s is waiting for user input\n", taskID)
	case execution.RunStatusFailed:
		if err := execution.UpdateTaskStatus(path, taskID, plan.StatusFailed); err != nil {
			return err
		}
		if execErr != nil {
			fmt.Fprintf(os.Stdout, "failed %s: %v\n", taskID, execErr)
		} else {
			fmt.Fprintf(os.Stdout, "failed %s\n", taskID)
		}
	default:
		return fmt.Errorf("unexpected run status %q", record.Status)
	}

	return nil
}
