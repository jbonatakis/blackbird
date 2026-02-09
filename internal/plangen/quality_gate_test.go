package plangen

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunQualityGateParityForEquivalentRefineResponses(t *testing.T) {
	initial := testPlanWithDescription("TODO")
	stillBlocking := testPlanWithDescription("TBD")
	refined := testPlanWithDescription("Implement bounded plan quality gate orchestration with deterministic lint and shared callbacks.")

	cliRunner := newQueuedRunner([]agent.Response{
		{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			Plan:          &stillBlocking,
		},
		{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			Plan:          &refined,
		},
	})
	tuiRunner := newQueuedRunner([]agent.Response{
		{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			Plan:          &stillBlocking,
		},
		{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			Plan:          &refined,
		},
	})

	cliResult, err := RunQualityGate(context.Background(), initial, 3, refineWithRunner(cliRunner.Run))
	if err != nil {
		t.Fatalf("CLI-style RunQualityGate() error = %v", err)
	}
	tuiResult, err := RunQualityGate(context.Background(), initial, 3, refineWithRunner(tuiRunner.Run))
	if err != nil {
		t.Fatalf("TUI-style RunQualityGate() error = %v", err)
	}

	if cliResult.AutoRefinePassesRun != 2 {
		t.Fatalf("CLI-style passes = %d, want 2", cliResult.AutoRefinePassesRun)
	}
	if tuiResult.AutoRefinePassesRun != 2 {
		t.Fatalf("TUI-style passes = %d, want 2", tuiResult.AutoRefinePassesRun)
	}
	if !reflect.DeepEqual(cliResult.InitialFindings, tuiResult.InitialFindings) {
		t.Fatalf("initial findings mismatch:\ncli=%v\ntui=%v", cliResult.InitialFindings, tuiResult.InitialFindings)
	}
	if !reflect.DeepEqual(cliResult.FinalFindings, tuiResult.FinalFindings) {
		t.Fatalf("final findings mismatch:\ncli=%v\ntui=%v", cliResult.FinalFindings, tuiResult.FinalFindings)
	}
}

type queuedRunner struct {
	responses []agent.Response
	calls     int
}

func newQueuedRunner(responses []agent.Response) *queuedRunner {
	out := make([]agent.Response, len(responses))
	copy(out, responses)
	return &queuedRunner{responses: out}
}

func (r *queuedRunner) Run(_ context.Context, req agent.Request) (agent.Response, agent.Diagnostics, error) {
	if req.Type != agent.RequestPlanRefine {
		return agent.Response{}, agent.Diagnostics{}, errors.New("unexpected request type")
	}
	if r.calls >= len(r.responses) {
		return agent.Response{}, agent.Diagnostics{}, errors.New("unexpected extra refine call")
	}
	resp := r.responses[r.calls]
	r.calls++
	return resp, agent.Diagnostics{}, nil
}

func refineWithRunner(run Runner) QualityGateRefineFunc {
	return func(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) (plan.WorkGraph, error) {
		result, err := Refine(ctx, run, RefineInput{
			ChangeRequest: changeRequest,
			CurrentPlan:   currentPlan,
			Metadata: agent.RequestMetadata{
				JSONSchema: agent.DefaultPlanJSONSchema(),
			},
		})
		if err != nil {
			return plan.WorkGraph{}, err
		}
		if len(result.Questions) > 0 {
			return plan.WorkGraph{}, errors.New("unexpected clarification questions")
		}
		if result.Plan == nil {
			return plan.WorkGraph{}, errors.New("missing refined plan")
		}
		return plan.Clone(*result.Plan), nil
	}
}

func testPlanWithDescription(description string) plan.WorkGraph {
	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"leaf": {
				ID:          "leaf",
				Title:       "Implement quality gate",
				Description: description,
				AcceptanceCriteria: []string{
					"`go test ./...` passes.",
				},
				Prompt:   "Implement deterministic quality gate behavior and validate with tests.",
				ChildIDs: []string{},
				Deps:     []string{},
				Status:   plan.StatusTodo,
			},
		},
	}
}
