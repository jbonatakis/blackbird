package plangen

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

var ErrRunnerRequired = errors.New("plangen: runner is required")

// Runner executes a single agent request and returns the decoded response.
type Runner func(ctx context.Context, req agent.Request) (agent.Response, agent.Diagnostics, error)

// PlanResponseResult captures one plan request result.
// If Questions is non-empty, Plan will be nil.
type PlanResponseResult struct {
	Plan        *plan.WorkGraph
	Questions   []agent.Question
	Diagnostics agent.Diagnostics
}

// GenerateInput captures one plan_generate invocation.
type GenerateInput struct {
	Description string
	Constraints []string
	Granularity string
	Answers     []agent.Answer
	Metadata    agent.RequestMetadata
}

// RefineInput captures one plan_refine invocation.
type RefineInput struct {
	ChangeRequest string
	CurrentPlan   plan.WorkGraph
	Answers       []agent.Answer
	Metadata      agent.RequestMetadata
}

// Generate executes one plan_generate request and converts a non-question response into a plan.
func Generate(ctx context.Context, run Runner, input GenerateInput) (PlanResponseResult, error) {
	if run == nil {
		return PlanResponseResult{}, ErrRunnerRequired
	}

	req := agent.Request{
		SchemaVersion:      agent.SchemaVersion,
		Type:               agent.RequestPlanGenerate,
		SystemPrompt:       agent.DefaultPlanSystemPrompt(),
		ProjectDescription: strings.TrimSpace(input.Description),
		Constraints:        cloneStrings(input.Constraints),
		Granularity:        strings.TrimSpace(input.Granularity),
		Answers:            cloneAnswers(input.Answers),
		Metadata:           input.Metadata,
	}

	resp, diag, err := run(ctx, req)
	if err != nil {
		return PlanResponseResult{Diagnostics: diag}, err
	}

	return responseToPlanResult(plan.NewEmptyWorkGraph(), resp, diag)
}

// Refine executes one plan_refine request and converts a non-question response into a plan.
func Refine(ctx context.Context, run Runner, input RefineInput) (PlanResponseResult, error) {
	if run == nil {
		return PlanResponseResult{}, ErrRunnerRequired
	}

	current := plan.Clone(input.CurrentPlan)
	req := agent.Request{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		SystemPrompt:  agent.DefaultPlanSystemPrompt(),
		ChangeRequest: strings.TrimSpace(input.ChangeRequest),
		Plan:          &current,
		Answers:       cloneAnswers(input.Answers),
		Metadata:      input.Metadata,
	}

	resp, diag, err := run(ctx, req)
	if err != nil {
		return PlanResponseResult{Diagnostics: diag}, err
	}

	return responseToPlanResult(current, resp, diag)
}

func responseToPlanResult(base plan.WorkGraph, resp agent.Response, diag agent.Diagnostics) (PlanResponseResult, error) {
	if len(resp.Questions) > 0 {
		return PlanResponseResult{
			Questions:   cloneQuestions(resp.Questions),
			Diagnostics: diag,
		}, nil
	}

	next, err := agent.ResponseToPlan(base, resp, time.Now().UTC())
	if err != nil {
		return PlanResponseResult{Diagnostics: diag}, err
	}

	return PlanResponseResult{
		Plan:        ptrPlan(next),
		Diagnostics: diag,
	}, nil
}

func ptrPlan(g plan.WorkGraph) *plan.WorkGraph {
	out := plan.Clone(g)
	return &out
}

func cloneAnswers(in []agent.Answer) []agent.Answer {
	if len(in) == 0 {
		return nil
	}
	out := make([]agent.Answer, len(in))
	copy(out, in)
	return out
}

func cloneQuestions(in []agent.Question) []agent.Question {
	if len(in) == 0 {
		return nil
	}
	out := make([]agent.Question, len(in))
	for i := range in {
		out[i] = agent.Question{
			ID:      in[i].ID,
			Prompt:  in[i].Prompt,
			Options: cloneStrings(in[i].Options),
		}
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
