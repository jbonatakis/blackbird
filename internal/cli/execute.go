package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
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

	result, err := execution.RunExecute(ctx, execution.ExecuteConfig{
		PlanPath: path,
		Runtime:  runtime,
		OnTaskStart: func(taskID string) {
			fmt.Fprintf(os.Stdout, "starting %s\n", taskID)
		},
		OnTaskFinish: func(taskID string, record execution.RunRecord, execErr error) {
			switch record.Status {
			case execution.RunStatusSuccess:
				fmt.Fprintf(os.Stdout, "completed %s\n", taskID)
			case execution.RunStatusFailed:
				if execErr != nil {
					fmt.Fprintf(os.Stdout, "failed %s: %v\n", taskID, execErr)
				} else {
					fmt.Fprintf(os.Stdout, "failed %s\n", taskID)
				}
			}
		},
	})
	if err != nil {
		return err
	}

	switch result.Reason {
	case execution.ExecuteReasonCompleted:
		fmt.Fprintln(os.Stdout, "no ready tasks remaining")
	case execution.ExecuteReasonWaitingUser:
		if result.TaskID != "" {
			fmt.Fprintf(os.Stdout, "%s is waiting for user input\n", result.TaskID)
		} else {
			fmt.Fprintln(os.Stdout, "waiting for user input")
		}
	case execution.ExecuteReasonCanceled:
		fmt.Fprintln(os.Stdout, "execution interrupted")
	case execution.ExecuteReasonError:
		if result.Err != nil {
			return result.Err
		}
		return fmt.Errorf("execution stopped with error")
	}

	return nil
}
