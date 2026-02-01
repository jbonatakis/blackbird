package agent

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestResponseToPlanNormalizesFullPlanTimestamps(t *testing.T) {
	now := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	agentTime := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	parentID := "parent"
	childID := "child"

	resp := Response{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanGenerate,
		Plan: &plan.WorkGraph{
			SchemaVersion: plan.SchemaVersion,
			Items: map[string]plan.WorkItem{
				parentID: makeTestItem(parentID, agentTime, nil, []string{childID}),
				childID:  makeTestItem(childID, agentTime, &parentID, nil),
			},
		},
	}

	result, err := ResponseToPlan(plan.NewEmptyWorkGraph(), resp, now)
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

func TestResponseToPlanAppliesPatch(t *testing.T) {
	now := time.Date(2026, 1, 31, 10, 30, 0, 0, time.UTC)
	baseTime := time.Date(2026, 1, 31, 9, 30, 0, 0, time.UTC)

	base := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"root": makeTestItem("root", baseTime, nil, nil),
		},
	}

	parentID := "root"
	resp := Response{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanRefine,
		Patch: []PatchOp{
			{
				Op:       PatchAdd,
				ParentID: &parentID,
				Item: &plan.WorkItem{
					ID:                 "child",
					Title:              "Child",
					Description:        "",
					AcceptanceCriteria: []string{},
					Prompt:             "do it",
					ParentID:           nil,
					ChildIDs:           []string{},
					Deps:               []string{},
					Status:             plan.StatusTodo,
				},
			},
		},
	}

	result, err := ResponseToPlan(base, resp, now)
	if err != nil {
		t.Fatalf("responseToPlan: %v", err)
	}
	if errs := plan.Validate(result); len(errs) != 0 {
		t.Fatalf("plan validation failed: %v", errs)
	}

	child, ok := result.Items["child"]
	if !ok {
		t.Fatalf("expected child item after patch")
	}
	if !child.CreatedAt.Equal(now) || !child.UpdatedAt.Equal(now) {
		t.Fatalf("child timestamps not updated: got %s/%s want %s", child.CreatedAt, child.UpdatedAt, now)
	}
	if child.ParentID == nil || *child.ParentID != "root" {
		t.Fatalf("expected child parent root, got %v", child.ParentID)
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
