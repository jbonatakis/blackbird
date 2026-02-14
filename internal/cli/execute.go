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
	"strings"
	"syscall"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/config"
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

	path := plan.PlanPath()
	if _, err := loadValidatedPlan(path); err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig(filepath.Dir(path))
	if err != nil {
		cfg = config.DefaultResolvedConfig()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	controller := execution.ExecutionController{
		PlanPath:            path,
		Runtime:             runtime,
		StopAfterEachTask:   cfg.Execution.StopAfterEachTask,
		ParentReviewEnabled: cfg.Execution.ParentReviewEnabled,
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
	}

	result, err := controller.Execute(ctx)
	if err != nil {
		return err
	}

	return handleExecuteResult(ctx, controller, result)
}

func handleExecuteResult(ctx context.Context, controller execution.ExecutionController, result execution.ExecuteResult) error {
	for {
		switch result.Reason {
		case execution.ExecuteReasonCompleted:
			fmt.Fprintln(os.Stdout, "no ready tasks remaining")
			return nil
		case execution.ExecuteReasonWaitingUser:
			if result.TaskID != "" {
				fmt.Fprintf(os.Stdout, "%s is waiting for user input\n", result.TaskID)
			} else {
				fmt.Fprintln(os.Stdout, "waiting for user input")
			}
			return nil
		case execution.ExecuteReasonDecisionRequired:
			next, err := handleDecisionRequired(ctx, controller, result)
			if err != nil {
				return err
			}
			if next == nil {
				return nil
			}
			result = *next
			continue
		case execution.ExecuteReasonParentReviewRequired:
			printParentReviewRequired(os.Stdout, result.TaskID, result.Run)
			return nil
		case execution.ExecuteReasonCanceled:
			fmt.Fprintln(os.Stdout, "execution interrupted")
			return nil
		case execution.ExecuteReasonError:
			if result.Err != nil {
				return result.Err
			}
			return fmt.Errorf("execution stopped with error")
		default:
			return fmt.Errorf("unknown execution stop reason %q", result.Reason)
		}
	}
}

func handleDecisionRequired(ctx context.Context, controller execution.ExecutionController, result execution.ExecuteResult) (*execution.ExecuteResult, error) {
	taskID := strings.TrimSpace(result.TaskID)
	run := result.Run
	if taskID == "" && run != nil {
		taskID = run.TaskID
	}
	if taskID == "" {
		return nil, fmt.Errorf("decision required without task id")
	}

	if run == nil {
		latest, err := execution.GetLatestRun(filepath.Dir(controller.PlanPath), taskID)
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, fmt.Errorf("no runs found for %s", taskID)
		}
		run = latest
	}

	title := strings.TrimSpace(run.Context.Task.Title)
	if title == "" {
		g, err := loadValidatedPlan(controller.PlanPath)
		if err == nil {
			if item, ok := g.Items[taskID]; ok {
				title = item.Title
			}
		}
	}

	for {
		printReviewPrompt(os.Stdout, taskID, title, run.Status, run.ReviewSummary)
		option, err := promptReviewDecision(defaultReviewDecisionOptions())
		if err != nil {
			return nil, err
		}

		feedback := ""
		if option.RequiresFeedback {
			feedback, err = promptReviewFeedback()
			if err != nil {
				if errors.Is(err, errReviewPromptCanceled) {
					continue
				}
				return nil, err
			}
		}

		decision, err := controller.ResolveDecision(ctx, execution.DecisionRequest{
			TaskID:   taskID,
			RunID:    run.ID,
			Action:   option.Action,
			Feedback: feedback,
		})
		if err != nil {
			return nil, err
		}

		switch decision.Action {
		case execution.DecisionStateApprovedContinue:
			if decision.Next != nil {
				// Deferred parent-review failures (and any non-completed outcomes)
				// must pause before execute continues.
				if decision.Next.Reason != execution.ExecuteReasonCompleted || !decision.Continue {
					return decision.Next, nil
				}
			}
			if !decision.Continue {
				return nil, nil
			}
			next, err := controller.Execute(ctx)
			if err != nil {
				return nil, err
			}
			return &next, nil
		case execution.DecisionStateApprovedQuit:
			return nil, nil
		case execution.DecisionStateRejected:
			return nil, nil
		case execution.DecisionStateChangesRequested:
			if decision.Next != nil {
				return decision.Next, nil
			}
			return nil, nil
		default:
			return nil, fmt.Errorf("unsupported decision action %q", decision.Action)
		}
	}
}
