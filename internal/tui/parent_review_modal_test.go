package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestNewParentReviewFormNormalizesTargetsAndFeedback(t *testing.T) {
	parentID := "parent-checkout"
	now := time.Date(2026, 2, 9, 8, 30, 0, 0, time.UTC)

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: {
				ID:                 parentID,
				Title:              "Checkout Parent Review",
				AcceptanceCriteria: []string{"All checkout paths are validated."},
				Status:             plan.StatusDone,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	run := execution.RunRecord{
		ID:                        "review-42",
		TaskID:                    parentID,
		ParentReviewResumeTaskIDs: []string{" child-b ", "child-a", "child-b", ""},
		ParentReviewFeedback:      "  Missing validation coverage.\n\nRetry with stricter checks.  ",
		Context: execution.ContextPack{
			Task: execution.TaskContext{ID: parentID},
		},
	}

	form := NewParentReviewForm(run, g)

	if form.parentTask.ID != parentID {
		t.Fatalf("parentTask.ID = %q, want %q", form.parentTask.ID, parentID)
	}
	if form.parentTask.Title != "Checkout Parent Review" {
		t.Fatalf("parentTask.Title = %q, want %q", form.parentTask.Title, "Checkout Parent Review")
	}
	if got, want := form.ResumeTargets(), []string{"child-a", "child-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ResumeTargets() = %#v, want %#v", got, want)
	}
	if form.selectedTarget != 0 {
		t.Fatalf("selectedTarget = %d, want 0", form.selectedTarget)
	}
	if got := form.Feedback(); got != "Missing validation coverage.\nRetry with stricter checks." {
		t.Fatalf("Feedback() = %q", got)
	}
}

func TestParentReviewFormUpdateNavigatesTargets(t *testing.T) {
	form := NewParentReviewForm(testParentReviewRun(), plan.NewEmptyWorkGraph())

	if form.SelectedTarget() != "child-a" {
		t.Fatalf("initial SelectedTarget() = %q, want child-a", form.SelectedTarget())
	}

	var action ParentReviewModalAction
	form, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("action after down = %v, want none", action)
	}
	if form.SelectedTarget() != "child-b" {
		t.Fatalf("SelectedTarget() after down = %q, want child-b", form.SelectedTarget())
	}

	form, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("action after second down = %v, want none", action)
	}
	if form.SelectedTarget() != "child-c" {
		t.Fatalf("SelectedTarget() after second down = %q, want child-c", form.SelectedTarget())
	}

	form, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("action after third down = %v, want none", action)
	}
	if form.SelectedTarget() != "child-c" {
		t.Fatalf("SelectedTarget() should stay at last target, got %q", form.SelectedTarget())
	}

	form, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("action after up = %v, want none", action)
	}
	if form.SelectedTarget() != "child-b" {
		t.Fatalf("SelectedTarget() after up = %q, want child-b", form.SelectedTarget())
	}
}

func TestParentReviewFormUpdateReturnsExplicitActions(t *testing.T) {
	form := NewParentReviewForm(testParentReviewRun(), plan.NewEmptyWorkGraph())
	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	_, action := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionResumeSelected {
		t.Fatalf("enter action = %v, want resume selected", action)
	}

	_, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if action != ParentReviewModalActionResumeSelected {
		t.Fatalf("1 action = %v, want resume selected", action)
	}

	_, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if action != ParentReviewModalActionResumeAll {
		t.Fatalf("2 action = %v, want resume all", action)
	}

	_, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if action != ParentReviewModalActionDismiss {
		t.Fatalf("3 action = %v, want dismiss", action)
	}

	_, action = form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if action != ParentReviewModalActionDismiss {
		t.Fatalf("esc action = %v, want dismiss", action)
	}
}

func TestParentReviewFormUpdateNoTargetsDisablesResumeActions(t *testing.T) {
	run := testParentReviewRun()
	run.ParentReviewResumeTaskIDs = nil
	form := NewParentReviewForm(run, plan.NewEmptyWorkGraph())

	if form.SelectedTarget() != "" {
		t.Fatalf("SelectedTarget() = %q, want empty", form.SelectedTarget())
	}

	_, action := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionNone {
		t.Fatalf("enter action with no targets = %v, want none", action)
	}

	_, action = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("2 action with no targets = %v, want none", action)
	}
}

func TestRenderParentReviewModalIncludesDeterministicContent(t *testing.T) {
	run := testParentReviewRun()
	m := Model{windowWidth: 100, windowHeight: 30}
	form := NewParentReviewForm(run, plan.NewEmptyWorkGraph())

	first := stripANSI(RenderParentReviewModal(m, form))
	second := stripANSI(RenderParentReviewModal(m, form))
	if first != second {
		t.Fatalf("expected deterministic render output")
	}

	for _, want := range []string{
		"Parent review failed",
		"Parent task: parent-checkout - Parent Checkout Review",
		"Review run: review-42",
		"Outcome: failed",
		"Resume targets:",
		"1. child-a",
		"2. child-b",
		"3. child-c",
		"Feedback:",
		"Child outputs miss required validation paths.",
		"Retry with stricter coverage.",
		"Actions:",
		"1. Resume selected target",
		"2. Resume all targets",
		"3. Dismiss",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("expected modal to include %q, got:\n%s", want, first)
		}
	}

	childAIdx := strings.Index(first, "1. child-a")
	childBIdx := strings.Index(first, "2. child-b")
	childCIdx := strings.Index(first, "3. child-c")
	if childAIdx == -1 || childBIdx == -1 || childCIdx == -1 {
		t.Fatalf("expected all sorted resume targets in output")
	}
	if !(childAIdx < childBIdx && childBIdx < childCIdx) {
		t.Fatalf("expected sorted resume target order, got:\n%s", first)
	}
}

func testParentReviewRun() execution.RunRecord {
	passed := false
	return execution.RunRecord{
		ID:                 "review-42",
		TaskID:             "parent-checkout",
		ParentReviewPassed: &passed,
		ParentReviewResumeTaskIDs: []string{
			"child-b",
			" child-c ",
			"child-a",
		},
		ParentReviewFeedback: "Child outputs miss required validation paths.\n\nRetry with stricter coverage.",
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-checkout",
				Title: "Parent Checkout Review",
			},
			ParentReview: &execution.ParentReviewContext{
				ParentTaskID:    "parent-checkout",
				ParentTaskTitle: "Parent Checkout Review",
			},
		},
	}
}
