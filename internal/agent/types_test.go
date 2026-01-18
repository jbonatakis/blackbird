package agent

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestValidateRequestGenerateRequiresDescription(t *testing.T) {
	req := Request{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanGenerate,
	}
	errs := ValidateRequest(req)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors")
	}
}

func TestValidateRequestRefineRequiresPlanAndChangeRequest(t *testing.T) {
	req := Request{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanRefine,
	}
	errs := ValidateRequest(req)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors")
	}
}

func TestValidateResponseQuestionsOnly(t *testing.T) {
	resp := Response{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanGenerate,
		Questions:     []Question{{ID: "q1", Prompt: "What?"}},
	}
	errs := ValidateResponse(resp)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidateResponsePatchOpRequiredFields(t *testing.T) {
	resp := Response{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanRefine,
		Patch: []PatchOp{
			{Op: PatchDelete},
		},
	}
	errs := ValidateResponse(resp)
	if len(errs) == 0 {
		t.Fatalf("expected errors for missing id")
	}
}

func TestValidatePatchAddRequiresValidItem(t *testing.T) {
	now := time.Now().UTC()
	item := plan.WorkItem{
		ID:                 "task-1",
		Title:              "Task",
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	resp := Response{
		SchemaVersion: SchemaVersion,
		Type:          RequestPlanGenerate,
		Patch: []PatchOp{
			{Op: PatchAdd, Item: &item},
		},
	}
	errs := ValidateResponse(resp)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}
