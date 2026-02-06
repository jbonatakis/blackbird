package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
	"github.com/jbonatakis/blackbird/internal/plangen"
	"github.com/jbonatakis/blackbird/internal/planquality"
)

type PlanActionComplete struct {
	Action  string
	Success bool
	Output  string
	Err     error
}

type ExecuteActionComplete struct {
	Action  string
	Success bool
	Output  string
	Err     error
	Result  *execution.ExecuteResult
	Record  *execution.RunRecord
}

type DecisionActionComplete struct {
	Action execution.DecisionState
	Result execution.DecisionResult
	Err    error
}

type PlanGenerateInMemoryResult struct {
	Success   bool
	Plan      *plan.WorkGraph
	Quality   *PlanReviewQualitySummary
	Questions []agent.Question
	Err       error
}

func ExecuteCmd() tea.Cmd {
	ctx := context.Background()
	return ExecuteCmdWithContext(ctx)
}

func ExecuteCmdWithContext(ctx context.Context) tea.Cmd {
	return ExecuteCmdWithContextAndStream(ctx, nil, nil, nil, false)
}

func ExecuteCmdWithContextAndStream(ctx context.Context, stdout io.Writer, stderr io.Writer, liveOutput chan liveOutputMsg, stopAfterEachTask bool) tea.Cmd {
	return func() tea.Msg {
		if liveOutput != nil {
			defer close(liveOutput)
		}
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return ExecuteActionComplete{Action: "execute", Success: false, Err: err}
		}

		result, runErr := execution.RunExecute(ctx, execution.ExecuteConfig{
			PlanPath:          plan.PlanPath(),
			Runtime:           runtime,
			StopAfterEachTask: stopAfterEachTask,
			StreamStdout:      stdout,
			StreamStderr:      stderr,
		})
		msg := ExecuteActionComplete{
			Action: "execute",
			Result: &result,
			Err:    runErr,
		}
		if runErr != nil {
			return msg
		}
		msg.Success = true
		msg.Output = summarizeExecuteResult(result)
		return msg
	}
}

func ResumeCmd(taskID string, answers []agent.Answer) tea.Cmd {
	ctx := context.Background()
	return ResumeCmdWithContext(ctx, taskID, answers)
}

func ResumeCmdWithContext(ctx context.Context, taskID string, answers []agent.Answer) tea.Cmd {
	return ResumeCmdWithContextAndStream(ctx, taskID, answers, nil, nil, nil)
}

func ResumeCmdWithContextAndStream(ctx context.Context, taskID string, answers []agent.Answer, stdout io.Writer, stderr io.Writer, liveOutput chan liveOutputMsg) tea.Cmd {
	return func() tea.Msg {
		if liveOutput != nil {
			defer close(liveOutput)
		}
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return ExecuteActionComplete{Action: "resume", Success: false, Err: err}
		}

		record, runErr := execution.RunResume(ctx, execution.ResumeConfig{
			PlanPath:     plan.PlanPath(),
			TaskID:       taskID,
			Answers:      answers,
			Runtime:      runtime,
			StreamStdout: stdout,
			StreamStderr: stderr,
		})
		msg := ExecuteActionComplete{
			Action: "resume",
			Record: &record,
			Err:    runErr,
		}
		if runErr != nil {
			return msg
		}
		msg.Success = true
		msg.Output = summarizeResumeRecord(record, taskID)
		return msg
	}
}

func ResolveDecisionCmdWithContext(ctx context.Context, taskID string, runID string, action execution.DecisionState, feedback string, stopAfterEachTask bool) tea.Cmd {
	return ResolveDecisionCmdWithContextAndStream(ctx, taskID, runID, action, feedback, stopAfterEachTask, nil, nil, nil)
}

func ResolveDecisionCmdWithContextAndStream(ctx context.Context, taskID string, runID string, action execution.DecisionState, feedback string, stopAfterEachTask bool, stdout io.Writer, stderr io.Writer, liveOutput chan liveOutputMsg) tea.Cmd {
	return func() tea.Msg {
		if liveOutput != nil {
			defer close(liveOutput)
		}
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return DecisionActionComplete{Action: action, Err: err}
		}

		controller := execution.ExecutionController{
			PlanPath:          plan.PlanPath(),
			Runtime:           runtime,
			StopAfterEachTask: stopAfterEachTask,
			StreamStdout:      stdout,
			StreamStderr:      stderr,
		}

		result, err := controller.ResolveDecision(ctx, execution.DecisionRequest{
			TaskID:   taskID,
			RunID:    runID,
			Action:   action,
			Feedback: feedback,
		})
		if err != nil {
			return DecisionActionComplete{Action: action, Err: err}
		}

		return DecisionActionComplete{Action: action, Result: result}
	}
}

func SetStatusCmd(id string, status string) tea.Cmd {
	return func() tea.Msg {
		s, ok := plan.ParseStatus(status)
		if !ok {
			return ExecuteActionComplete{
				Action: "set-status",
				Err:    fmt.Errorf("invalid status %q", status),
			}
		}
		path := plan.PlanPath()
		g, err := plan.Load(path)
		if err != nil {
			if errors.Is(err, plan.ErrPlanNotFound) {
				return ExecuteActionComplete{
					Action: "set-status",
					Err:    fmt.Errorf("plan file not found: %s (run `blackbird init`)", path),
				}
			}
			return ExecuteActionComplete{Action: "set-status", Err: err}
		}
		if errs := plan.Validate(g); len(errs) != 0 {
			return ExecuteActionComplete{
				Action: "set-status",
				Err:    fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path),
			}
		}

		now := time.Now().UTC()
		if err := plan.SetStatus(&g, id, s, now); err != nil {
			return ExecuteActionComplete{Action: "set-status", Err: err}
		}
		if err := plan.SaveAtomic(path, g); err != nil {
			return ExecuteActionComplete{Action: "set-status", Err: fmt.Errorf("write plan file: %w", err)}
		}
		return ExecuteActionComplete{
			Action:  "set-status",
			Success: true,
			Output:  fmt.Sprintf("updated %s status to %s\n", id, s),
		}
	}
}

func summarizeExecuteResult(result execution.ExecuteResult) string {
	switch result.Reason {
	case execution.ExecuteReasonCompleted:
		return "no ready tasks remaining"
	case execution.ExecuteReasonWaitingUser:
		if result.TaskID != "" {
			return result.TaskID + " is waiting for user input"
		}
		return "waiting for user input"
	case execution.ExecuteReasonDecisionRequired:
		if result.TaskID != "" {
			return result.TaskID + " requires review before continuing"
		}
		return "decision required before continuing"
	case execution.ExecuteReasonCanceled:
		return "execution interrupted"
	case execution.ExecuteReasonError:
		if result.Err != nil {
			return result.Err.Error()
		}
		return "execution stopped with error"
	default:
		return "execution finished"
	}
}

func summarizeResumeRecord(record execution.RunRecord, taskID string) string {
	switch record.Status {
	case execution.RunStatusSuccess:
		return "completed " + taskID
	case execution.RunStatusWaitingUser:
		return taskID + " is waiting for user input"
	case execution.RunStatusFailed:
		if record.Error != "" {
			return "failed " + taskID + ": " + record.Error
		}
		return "failed " + taskID
	default:
		return "resume finished"
	}
}

// GeneratePlanInMemory invokes the agent runtime directly without spawning a subprocess
// to generate a plan. It accepts the three input parameters (description, constraints, granularity)
// and calls the agent API without interactive prompting.
//
// If the agent asks questions, they are stored in the result for TUI display but the execution
// stops (does not auto-answer). The caller can then display questions to the user and call
// GeneratePlanInMemoryWithAnswers to continue with answers.
func GeneratePlanInMemory(ctx context.Context, description string, constraints []string, granularity string) tea.Cmd {
	return func() tea.Msg {
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		generateResult, err := plangen.Generate(ctx, runtime.Run, plangen.GenerateInput{
			Description: strings.TrimSpace(description),
			Constraints: trimNonEmpty(constraints),
			Granularity: strings.TrimSpace(granularity),
			Metadata:    requestMeta,
		})
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		if len(generateResult.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: generateResult.Questions,
			}
		}
		if generateResult.Plan == nil {
			return PlanGenerateInMemoryResult{Success: false, Err: errors.New("plan generation returned no plan")}
		}

		resultPlan := plan.Clone(*generateResult.Plan)
		qualityResult, err := runGeneratedPlanQualityGate(ctx, runtime, requestMeta, resultPlan)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		qualitySummary := buildPlanReviewQualitySummary(qualityResult)

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &qualityResult.FinalPlan,
			Quality: &qualitySummary,
		}
	}
}

// GeneratePlanInMemoryWithAnswers continues plan generation after answering agent questions.
// It takes the original request parameters plus the answers to questions that were asked.
func GeneratePlanInMemoryWithAnswers(ctx context.Context, description string, constraints []string, granularity string, answers []agent.Answer) tea.Cmd {
	return func() tea.Msg {
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		generateResult, err := plangen.Generate(ctx, runtime.Run, plangen.GenerateInput{
			Description: strings.TrimSpace(description),
			Constraints: trimNonEmpty(constraints),
			Granularity: strings.TrimSpace(granularity),
			Answers:     answers,
			Metadata:    requestMeta,
		})
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		if len(generateResult.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: generateResult.Questions,
			}
		}
		if generateResult.Plan == nil {
			return PlanGenerateInMemoryResult{Success: false, Err: errors.New("plan generation returned no plan")}
		}

		resultPlan := plan.Clone(*generateResult.Plan)
		qualityResult, err := runGeneratedPlanQualityGate(ctx, runtime, requestMeta, resultPlan)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		qualitySummary := buildPlanReviewQualitySummary(qualityResult)

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &qualityResult.FinalPlan,
			Quality: &qualitySummary,
		}
	}
}

// RefinePlanInMemory refines an existing plan with a change request
func RefinePlanInMemory(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) tea.Cmd {
	return func() tea.Msg {
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		refineResult, err := plangen.Refine(ctx, runtime.Run, plangen.RefineInput{
			ChangeRequest: strings.TrimSpace(changeRequest),
			CurrentPlan:   currentPlan,
			Metadata:      requestMeta,
		})
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		if len(refineResult.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: refineResult.Questions,
			}
		}
		if refineResult.Plan == nil {
			return PlanGenerateInMemoryResult{Success: false, Err: errors.New("plan refine returned no plan")}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    refineResult.Plan,
		}
	}
}

// RefinePlanInMemoryWithAnswers continues plan refinement after answering questions.
func RefinePlanInMemoryWithAnswers(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph, answers []agent.Answer) tea.Cmd {
	return func() tea.Msg {
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		refineResult, err := plangen.Refine(ctx, runtime.Run, plangen.RefineInput{
			ChangeRequest: strings.TrimSpace(changeRequest),
			CurrentPlan:   currentPlan,
			Answers:       answers,
			Metadata:      requestMeta,
		})
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}
		if len(refineResult.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: refineResult.Questions,
			}
		}
		if refineResult.Plan == nil {
			return PlanGenerateInMemoryResult{Success: false, Err: errors.New("plan refine returned no plan")}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    refineResult.Plan,
		}
	}
}

// ContinuePlanGenerationWithAnswers continues plan generation after answering questions
func ContinuePlanGenerationWithAnswers(description string, constraints []string, granularity string, answers []agent.Answer, questionRound int) tea.Cmd {
	// Check if we've exceeded max question rounds
	if questionRound >= agent.MaxPlanQuestionRounds {
		return func() tea.Msg {
			return PlanGenerateInMemoryResult{
				Success: false,
				Err:     agent.RuntimeError{Message: "too many clarification rounds"},
			}
		}
	}

	return GeneratePlanInMemoryWithAnswers(context.Background(), description, constraints, granularity, answers)
}

// ContinuePlanRefineWithAnswers continues plan refinement after answering questions.
func ContinuePlanRefineWithAnswers(changeRequest string, currentPlan plan.WorkGraph, answers []agent.Answer, questionRound int) tea.Cmd {
	if questionRound >= agent.MaxPlanQuestionRounds {
		return func() tea.Msg {
			return PlanGenerateInMemoryResult{
				Success: false,
				Err:     agent.RuntimeError{Message: "too many clarification rounds"},
			}
		}
	}

	return RefinePlanInMemoryWithAnswers(context.Background(), changeRequest, currentPlan, answers)
}

// trimNonEmpty removes empty strings from a slice after trimming whitespace
func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func runGeneratedPlanQualityGate(ctx context.Context, runtime agent.Runtime, requestMeta agent.RequestMetadata, generated plan.WorkGraph) (planquality.QualityGateResult, error) {
	maxPasses := plangen.ResolveMaxAutoRefinePassesFromPlanPath(plan.PlanPath())
	return plangen.RunQualityGate(ctx, generated, maxPasses, func(refineCtx context.Context, changeRequest string, currentPlan plan.WorkGraph) (plan.WorkGraph, error) {
		refineResult, err := plangen.Refine(refineCtx, runtime.Run, plangen.RefineInput{
			ChangeRequest: changeRequest,
			CurrentPlan:   currentPlan,
			Metadata:      requestMeta,
		})
		if err != nil {
			return plan.WorkGraph{}, err
		}
		if len(refineResult.Questions) > 0 {
			return plan.WorkGraph{}, errors.New("quality auto-refine requested clarification")
		}
		if refineResult.Plan == nil {
			return plan.WorkGraph{}, errors.New("quality auto-refine returned no plan")
		}
		return plan.Clone(*refineResult.Plan), nil
	})
}
