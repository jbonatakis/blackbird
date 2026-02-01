package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runResume(taskID string) error {
	if taskID == "" {
		return UsageError{Message: "resume requires exactly 1 argument: <taskID>"}
	}

	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}
	if _, ok := g.Items[taskID]; !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runs, err := execution.ListRuns(filepath.Dir(path), taskID)
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
		fmt.Fprintf(os.Stdout, "no questions found in waiting run for %s\n", taskID)
		return nil
	}

	answers, err := promptAnswers(questions)
	if err != nil {
		return err
	}

	ctxPack, err := execution.ResumeWithAnswer(*waiting, answers)
	if err != nil {
		return err
	}

	record, err := execution.RunResume(ctx, execution.ResumeConfig{
		PlanPath: path,
		Graph:    &g,
		TaskID:   taskID,
		Answers:  answers,
		Context:  &ctxPack,
		Runtime:  runtime,
	})
	if err != nil {
		return err
	}

	switch record.Status {
	case execution.RunStatusSuccess:
		fmt.Fprintf(os.Stdout, "completed %s\n", taskID)
	case execution.RunStatusWaitingUser:
		fmt.Fprintf(os.Stdout, "%s is waiting for user input\n", taskID)
	case execution.RunStatusFailed:
		if record.Error != "" {
			fmt.Fprintf(os.Stdout, "failed %s: %s\n", taskID, record.Error)
		} else {
			fmt.Fprintf(os.Stdout, "failed %s\n", taskID)
		}
	default:
		return fmt.Errorf("unexpected run status %q", record.Status)
	}

	return nil
}
