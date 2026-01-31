package cli

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestResponseToPlanNormalizesFullPlanTimestamps(t *testing.T) {
	now := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	agentTime := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	parentID := "parent"
	childID := "child"

	resp := agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan: &plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				parentID: makeTestItem(parentID, agentTime, nil, []string{childID}),
				childID:  makeTestItem(childID, agentTime, &parentID, nil),
			},
		},
	}

	result, err := responseToPlan(plan.NewEmptyWorkGraph(), resp, now)
	if err != nil {
		t.Fatalf("responseToPlan: %v", err)
	}
	if errs := plan.Validate(result); len(errs) != 0 {
		t.Fatalf("plan validation failed: %v", errs)
	}

	for id, item := range result.Items {
		if !item.CreatedAt.Equal(now) || !item.UpdatedAt.Equal(now) {
			t.Fatalf("%s timestamps not normalized: got %s/%s want %s", id, item.CreatedAt, item.UpdatedAt, now)
		}
	}
}

func makeTestItem(id string, ts time.Time, parentID *string, childIDs []string) plan.WorkItem {
	if childIDs == nil {
		childIDs = []string{}
	}
	return plan.WorkItem{
		ID:                 id,
		Title:              "Task " + id,
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "do it",
		ParentID:           parentID,
		ChildIDs:           childIDs,
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          ts,
		UpdatedAt:          ts,
	}
}
