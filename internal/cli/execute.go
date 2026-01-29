package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runExecute(args []string) error {
	fs := flag.NewFlagSet("execute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "execute takes no positional arguments"}
	}

	path := planPath()
	if _, err := loadValidatedPlan(path); err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	baseDir := filepath.Dir(path)

	for {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stdout, "execution interrupted")
			return nil
		}

		g, err := loadValidatedPlan(path)
		if err != nil {
			return err
		}

		ready := execution.ReadyTasks(g)
		if len(ready) == 0 {
			fmt.Fprintln(os.Stdout, "no ready tasks remaining")
			return nil
		}

		taskID := ready[0]
		fmt.Fprintf(os.Stdout, "starting %s\n", taskID)

		ctxPack, err := execution.BuildContext(g, taskID)
		if err != nil {
			return err
		}

		if err := execution.UpdateTaskStatus(path, taskID, plan.StatusInProgress); err != nil {
			return err
		}

		record, execErr := execution.LaunchAgent(ctx, runtime, ctxPack)
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
			return nil
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

		if execErr != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return execErr
			}
			continue
		}
	}
}
