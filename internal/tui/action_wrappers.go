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
			PlanPath:     planPath(),
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
			PlanPath:     planPath(),
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
		path := planPath()
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
			JSONSchema: defaultPlanJSONSchema(),
		}

		// Build the agent request
		req := agent.Request{
			SchemaVersion:      agent.SchemaVersion,
			Type:               agent.RequestPlanGenerate,
			SystemPrompt:       defaultPlanSystemPrompt(),
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
			JSONSchema: defaultPlanJSONSchema(),
		}

		// Build the agent request with answers
		req := agent.Request{
			SchemaVersion:      agent.SchemaVersion,
			Type:               agent.RequestPlanGenerate,
			SystemPrompt:       defaultPlanSystemPrompt(),
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
			JSONSchema: defaultPlanJSONSchema(),
		}

		// Build the agent request
		req := agent.Request{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			SystemPrompt:  defaultPlanSystemPrompt(),
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
			JSONSchema: defaultPlanJSONSchema(),
		}

		// Build the agent request with answers
		req := agent.Request{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			SystemPrompt:  defaultPlanSystemPrompt(),
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

// defaultPlanJSONSchema returns the JSON schema for plan generation
func defaultPlanJSONSchema() string {
	return strings.TrimSpace(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["schemaVersion", "type"],
  "properties": {
    "schemaVersion": { "type": "integer" },
    "type": { "type": "string", "enum": ["plan_generate", "plan_refine", "deps_infer"] },
    "plan": { "$ref": "#/definitions/workGraph" },
    "patch": { "type": "array", "items": { "$ref": "#/definitions/patchOp" } },
    "questions": { "type": "array", "items": { "$ref": "#/definitions/question" } }
  },
  "oneOf": [
    { "required": ["plan"] },
    { "required": ["patch"] },
    { "required": ["questions"] }
  ],
  "definitions": {
    "workGraph": {
      "type": "object",
      "required": ["schemaVersion", "items"],
      "properties": {
        "schemaVersion": { "type": "integer" },
        "items": {
          "type": "object",
          "additionalProperties": { "$ref": "#/definitions/workItem" }
        }
      }
    },
    "workItem": {
      "type": "object",
      "required": [
        "id", "title", "description", "acceptanceCriteria", "prompt",
        "parentId", "childIds", "deps", "status", "createdAt", "updatedAt"
      ],
      "properties": {
        "id": { "type": "string" },
        "title": { "type": "string" },
        "description": { "type": "string" },
        "acceptanceCriteria": { "type": "array", "items": { "type": "string" } },
        "prompt": { "type": "string" },
        "parentId": { "type": ["string", "null"] },
        "childIds": { "type": "array", "items": { "type": "string" } },
        "deps": { "type": "array", "items": { "type": "string" } },
        "status": { "type": "string", "enum": ["todo", "in_progress", "blocked", "done", "skipped"] },
        "createdAt": { "type": "string", "format": "date-time" },
        "updatedAt": { "type": "string", "format": "date-time" },
        "notes": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "patchOp": {
      "type": "object",
      "required": ["op"],
      "properties": {
        "op": { "type": "string", "enum": ["add", "update", "delete", "move", "set_deps", "add_dep", "remove_dep"] },
        "id": { "type": "string" },
        "item": { "$ref": "#/definitions/workItem" },
        "parentId": { "type": ["string", "null"] },
        "index": { "type": "integer", "minimum": 0 },
        "deps": { "type": "array", "items": { "type": "string" } },
        "depId": { "type": "string" },
        "rationale": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "question": {
      "type": "object",
      "required": ["id", "prompt"],
      "properties": {
        "id": { "type": "string" },
        "prompt": { "type": "string" },
        "options": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}`)
}

// defaultPlanSystemPrompt returns the system prompt for plan generation
func defaultPlanSystemPrompt() string {
	return strings.TrimSpace("You are a planning agent for blackbird.\n\n" +
		"Return exactly one JSON object on stdout (or a single fenced ```json block).\n" +
		"Do not include any other text outside the JSON.\n\n" +
		"Response shape:\n" +
		"- Must include schemaVersion and type.\n" +
		"- Must include exactly one of: plan, patch, or questions.\n\n" +
		"Plan requirements:\n" +
		"- Plan must conform to the WorkGraph schema.\n" +
		"- Every WorkItem must include required fields: id, title, description, acceptanceCriteria, prompt, parentId, childIds, deps, status, createdAt, updatedAt.\n" +
		"- Use stable, unique ids and keep parent/child relationships consistent.\n" +
		"- Deps must reference existing ids and must not form cycles.\n\n" +
		"- Avoid meta tasks like \"design the app\" or \"plan the work\" unless explicitly requested; the plan itself is the design.\n" +
		"- Top-level features should be meaningful deliverables, not a generic \"root\" placeholder.\n\n" +
		"Patch requirements:\n" +
		"- Use only ops: add, update, delete, move, set_deps, add_dep, remove_dep.\n" +
		"- Include required fields for each op.\n" +
		"- Do not introduce cycles or invalid references.\n\n" +
		"Questions:\n" +
		"- If clarification is required, respond with questions only (no plan/patch).\n" +
		"- Each question must include id and prompt; options are optional.\n")
}

// ContinuePlanGenerationWithAnswers continues plan generation after answering questions
func ContinuePlanGenerationWithAnswers(description string, constraints []string, granularity string, answers []agent.Answer, questionRound int) tea.Cmd {
	// Check if we've exceeded max question rounds
	const maxAgentQuestionRounds = 2
	if questionRound >= maxAgentQuestionRounds {
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
	const maxAgentQuestionRounds = 2
	if questionRound >= maxAgentQuestionRounds {
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
