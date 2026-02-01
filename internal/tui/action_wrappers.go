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

type PlanGenerateInMemoryResult struct {
	Success   bool
	Plan      *plan.WorkGraph
	Questions []agent.Question
	Err       error
}

func ExecuteCmd() tea.Cmd {
	ctx := context.Background()
	return ExecuteCmdWithContext(ctx)
}

func ExecuteCmdWithContext(ctx context.Context) tea.Cmd {
	return ExecuteCmdWithContextAndStream(ctx, nil, nil, nil)
}

func ExecuteCmdWithContextAndStream(ctx context.Context, stdout io.Writer, stderr io.Writer, liveOutput chan liveOutputMsg) tea.Cmd {
	return func() tea.Msg {
		if liveOutput != nil {
			defer close(liveOutput)
		}
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return ExecuteActionComplete{Action: "execute", Success: false, Err: err}
		}

		result, runErr := execution.RunExecute(ctx, execution.ExecuteConfig{
			PlanPath:     plan.PlanPath(),
			Runtime:      runtime,
			StreamStdout: stdout,
			StreamStderr: stderr,
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
		// Create agent runtime from environment
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Prepare request metadata with JSON schema
		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		// Build the agent request
		req := agent.Request{
			SchemaVersion:      agent.SchemaVersion,
			Type:               agent.RequestPlanGenerate,
			SystemPrompt:       agent.DefaultPlanSystemPrompt(),
			ProjectDescription: strings.TrimSpace(description),
			Constraints:        trimNonEmpty(constraints),
			Granularity:        strings.TrimSpace(granularity),
			Metadata:           requestMeta,
		}

		// Run the agent (without interactive question loop)
		resp, _, err := runtime.Run(ctx, req)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Check if agent is asking questions
		if len(resp.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: resp.Questions,
			}
		}

		// Convert response to plan
		resultPlan, err := agent.ResponseToPlan(plan.NewEmptyWorkGraph(), resp, time.Now().UTC())
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &resultPlan,
		}
	}
}

// GeneratePlanInMemoryWithAnswers continues plan generation after answering agent questions.
// It takes the original request parameters plus the answers to questions that were asked.
func GeneratePlanInMemoryWithAnswers(ctx context.Context, description string, constraints []string, granularity string, answers []agent.Answer) tea.Cmd {
	return func() tea.Msg {
		// Create agent runtime from environment
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Prepare request metadata with JSON schema
		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		// Build the agent request with answers
		req := agent.Request{
			SchemaVersion:      agent.SchemaVersion,
			Type:               agent.RequestPlanGenerate,
			SystemPrompt:       agent.DefaultPlanSystemPrompt(),
			ProjectDescription: strings.TrimSpace(description),
			Constraints:        trimNonEmpty(constraints),
			Granularity:        strings.TrimSpace(granularity),
			Answers:            answers,
			Metadata:           requestMeta,
		}

		// Run the agent
		resp, _, err := runtime.Run(ctx, req)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Check if agent is asking MORE questions (limit rounds to prevent infinite loop)
		if len(resp.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: resp.Questions,
			}
		}

		// Convert response to plan
		resultPlan, err := agent.ResponseToPlan(plan.NewEmptyWorkGraph(), resp, time.Now().UTC())
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &resultPlan,
		}
	}
}

// RefinePlanInMemory refines an existing plan with a change request
func RefinePlanInMemory(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) tea.Cmd {
	return func() tea.Msg {
		// Create agent runtime from environment
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Prepare request metadata with JSON schema
		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		// Build the agent request
		req := agent.Request{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			SystemPrompt:  agent.DefaultPlanSystemPrompt(),
			ChangeRequest: strings.TrimSpace(changeRequest),
			Plan:          &currentPlan,
			Metadata:      requestMeta,
		}

		// Run the agent
		resp, _, err := runtime.Run(ctx, req)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Check if agent is asking questions
		if len(resp.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: resp.Questions,
			}
		}

		// Convert response to plan
		resultPlan, err := agent.ResponseToPlan(currentPlan, resp, time.Now().UTC())
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &resultPlan,
		}
	}
}

// RefinePlanInMemoryWithAnswers continues plan refinement after answering questions.
func RefinePlanInMemoryWithAnswers(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph, answers []agent.Answer) tea.Cmd {
	return func() tea.Msg {
		// Create agent runtime from environment
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Prepare request metadata with JSON schema
		requestMeta := agent.RequestMetadata{
			JSONSchema: agent.DefaultPlanJSONSchema(),
		}
		requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)

		// Build the agent request with answers
		req := agent.Request{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			SystemPrompt:  agent.DefaultPlanSystemPrompt(),
			ChangeRequest: strings.TrimSpace(changeRequest),
			Plan:          &currentPlan,
			Answers:       answers,
			Metadata:      requestMeta,
		}

		// Run the agent
		resp, _, err := runtime.Run(ctx, req)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Check if agent is asking MORE questions
		if len(resp.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: resp.Questions,
			}
		}

		// Convert response to plan
		resultPlan, err := agent.ResponseToPlan(currentPlan, resp, time.Now().UTC())
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &resultPlan,
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
