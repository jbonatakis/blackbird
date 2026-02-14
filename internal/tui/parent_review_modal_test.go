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
	if form.SelectedTarget() != "child-a" {
		t.Fatalf("SelectedTarget() = %q, want %q", form.SelectedTarget(), "child-a")
	}
	if form.SelectedAction() != parentReviewActionContinue {
		t.Fatalf("SelectedAction() = %d, want %d", form.SelectedAction(), parentReviewActionContinue)
	}
	if got := form.Feedback(); got != "Missing validation coverage.\nRetry with stricter checks." {
		t.Fatalf("Feedback() = %q", got)
	}
}

func TestNewParentReviewFormUsesStructuredResultsForTargetsAndFeedback(t *testing.T) {
	run := execution.RunRecord{
		ID:     "review-100",
		TaskID: "parent-checkout",
		ParentReviewResults: execution.ParentReviewTaskResults{
			"child-c": {
				TaskID: "child-c",
				Status: execution.ParentReviewTaskStatusPassed,
			},
			"child-b": {
				TaskID:   "child-b",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Fix child-b checkout retries.",
			},
			"child-a": {
				TaskID:   "child-a",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Fix child-a tax calculations.",
			},
		},
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-checkout",
				Title: "Parent Checkout Review",
			},
		},
	}

	form := NewParentReviewForm(run, plan.NewEmptyWorkGraph())
	if got, want := form.ReviewedTaskIDs(), []string{"child-a", "child-b", "child-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReviewedTaskIDs() = %#v, want %#v", got, want)
	}
	if got, want := form.ResumeTargets(), []string{"child-a", "child-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ResumeTargets() = %#v, want %#v", got, want)
	}
	if form.SelectedTarget() != "child-a" {
		t.Fatalf("SelectedTarget() = %q, want %q", form.SelectedTarget(), "child-a")
	}
	if form.Feedback() != "Fix child-a tax calculations." {
		t.Fatalf("Feedback() = %q", form.Feedback())
	}
}

func TestParentReviewFormUpdateMatchesInterruptionKeybindings(t *testing.T) {
	form := NewParentReviewForm(testParentReviewRun(), plan.NewEmptyWorkGraph())
	if form.SelectedAction() != parentReviewActionContinue {
		t.Fatalf("initial SelectedAction() = %d, want %d", form.SelectedAction(), parentReviewActionContinue)
	}

	updated, action := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("2 action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionResumeAllFailed {
		t.Fatalf("SelectedAction() after 2 = %d, want %d", updated.SelectedAction(), parentReviewActionResumeAllFailed)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionResumeAllFailed {
		t.Fatalf("enter action = %v, want resume all failed", action)
	}

	updated, action = form.Update(tea.KeyMsg{Type: tea.KeyDown})
	if action != ParentReviewModalActionNone {
		t.Fatalf("down action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionResumeAllFailed {
		t.Fatalf("SelectedAction() after down = %d, want %d", updated.SelectedAction(), parentReviewActionResumeAllFailed)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("j action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionResumeOneTask {
		t.Fatalf("SelectedAction() after j = %d, want %d", updated.SelectedAction(), parentReviewActionResumeOneTask)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("k action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionResumeAllFailed {
		t.Fatalf("SelectedAction() after k = %d, want %d", updated.SelectedAction(), parentReviewActionResumeAllFailed)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("3 action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionResumeOneTask {
		t.Fatalf("SelectedAction() after 3 = %d, want %d", updated.SelectedAction(), parentReviewActionResumeOneTask)
	}

	_, action = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionResumeOneTask {
		t.Fatalf("enter action on resume-one = %v, want resume one", action)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("4 action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionQuit {
		t.Fatalf("SelectedAction() after 4 = %d, want %d", updated.SelectedAction(), parentReviewActionQuit)
	}

	_, action = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionQuit {
		t.Fatalf("enter action on quit = %v, want quit", action)
	}

	_, action = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if action != ParentReviewModalActionContinue {
		t.Fatalf("esc action = %v, want continue", action)
	}
}

func TestParentReviewFormUpdateNoFailedDisablesResumeActions(t *testing.T) {
	form := NewParentReviewForm(testParentReviewAllPassedRun(), plan.NewEmptyWorkGraph())
	if form.HasFailedTasks() {
		t.Fatalf("expected HasFailedTasks() false")
	}

	updated, action := form.Update(tea.KeyMsg{Type: tea.KeyDown})
	if action != ParentReviewModalActionNone {
		t.Fatalf("down action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionQuit {
		t.Fatalf("SelectedAction() after down = %d, want %d", updated.SelectedAction(), parentReviewActionQuit)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	if action != ParentReviewModalActionNone {
		t.Fatalf("up action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionContinue {
		t.Fatalf("SelectedAction() after up = %d, want %d", updated.SelectedAction(), parentReviewActionContinue)
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("2 action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionContinue {
		t.Fatalf("SelectedAction() should remain continue when resume-all disabled, got %d", updated.SelectedAction())
	}

	updated, action = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if action != ParentReviewModalActionNone {
		t.Fatalf("3 action = %v, want none", action)
	}
	if updated.SelectedAction() != parentReviewActionContinue {
		t.Fatalf("SelectedAction() should remain continue when resume-one disabled, got %d", updated.SelectedAction())
	}

	updated.selectedAction = parentReviewActionResumeAllFailed
	_, action = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionNone {
		t.Fatalf("enter action on disabled resume-all = %v, want none", action)
	}

	updated.selectedAction = parentReviewActionResumeOneTask
	_, action = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action != ParentReviewModalActionNone {
		t.Fatalf("enter action on disabled resume-one = %v, want none", action)
	}
}

func TestRenderParentReviewModalRendersStructuredReviewContentAndActionOrder(t *testing.T) {
	run := testParentReviewRun()
	m := Model{windowWidth: 110, windowHeight: 34}
	form := NewParentReviewForm(run, plan.NewEmptyWorkGraph())

	first := stripANSI(RenderParentReviewModal(m, form))
	second := stripANSI(RenderParentReviewModal(m, form))
	if first != second {
		t.Fatalf("expected deterministic render output")
	}

	for _, want := range []string{
		"Post-review results",
		"Parent task: parent-checkout - Parent Checkout Review",
		"Review run: review-42",
		"Outcome: failed",
		"Reviewed tasks:",
		"[FAIL] child-a",
		"feedback: Fix child-a validation gaps.",
		"[FAIL] child-b",
		"feedback: Improve child-b retry handling.",
		"[PASS] child-c",
		"Actions:",
		"1. Continue",
		"2. Resume all failed",
		"3. Resume one task",
		"4. Quit",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("expected modal to include %q, got:\n%s", want, first)
		}
	}

	continueIdx := strings.Index(first, "1. Continue")
	resumeAllIdx := strings.Index(first, "2. Resume all failed")
	resumeOneIdx := strings.Index(first, "3. Resume one task")
	quitIdx := strings.Index(first, "4. Quit")
	if continueIdx == -1 || resumeAllIdx == -1 || resumeOneIdx == -1 || quitIdx == -1 {
		t.Fatalf("expected all four actions in output")
	}
	if !(continueIdx < resumeAllIdx && resumeAllIdx < resumeOneIdx && resumeOneIdx < quitIdx) {
		t.Fatalf("expected action order Continue -> Resume all -> Resume one -> Quit, got:\n%s", first)
	}
}

func TestRenderParentReviewModalNoFailedShowsDisabledResumeActionsAndExplanation(t *testing.T) {
	run := testParentReviewAllPassedRun()
	m := Model{windowWidth: 100, windowHeight: 30}
	form := NewParentReviewForm(run, plan.NewEmptyWorkGraph())

	out := stripANSI(RenderParentReviewModal(m, form))
	for _, want := range []string{
		"No failed tasks were reported; resume actions are disabled.",
		"2. Resume all failed (disabled)",
		"3. Resume one task (disabled)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected no-fail modal text %q, got:\n%s", want, out)
		}
	}
}

func TestParentReviewModalBorderColorReflectsAggregateResult(t *testing.T) {
	allPassed := NewParentReviewForm(testParentReviewAllPassedRun(), plan.NewEmptyWorkGraph())
	if got, want := string(parentReviewModalBorderColor(allPassed)), "46"; got != want {
		t.Fatalf("all-pass border color = %q, want %q", got, want)
	}

	mixed := NewParentReviewForm(testParentReviewRun(), plan.NewEmptyWorkGraph())
	if got, want := string(parentReviewModalBorderColor(mixed)), "214"; got != want {
		t.Fatalf("mixed border color = %q, want %q", got, want)
	}

	allFailed := NewParentReviewForm(testParentReviewAllFailedRun(), plan.NewEmptyWorkGraph())
	if got, want := string(parentReviewModalBorderColor(allFailed)), "196"; got != want {
		t.Fatalf("all-fail border color = %q, want %q", got, want)
	}
}

func testParentReviewRun() execution.RunRecord {
	passed := false
	return execution.RunRecord{
		ID:                 "review-42",
		TaskID:             "parent-checkout",
		ParentReviewPassed: &passed,
		ParentReviewResults: execution.ParentReviewTaskResults{
			"child-b": {
				TaskID:   "child-b",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Improve child-b retry handling.",
			},
			"child-c": {
				TaskID: "child-c",
				Status: execution.ParentReviewTaskStatusPassed,
			},
			"child-a": {
				TaskID:   "child-a",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Fix child-a validation gaps.",
			},
		},
		ParentReviewFeedback: "Fallback review feedback.",
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

func testParentReviewAllPassedRun() execution.RunRecord {
	passed := true
	return execution.RunRecord{
		ID:                 "review-77",
		TaskID:             "parent-checkout",
		ParentReviewPassed: &passed,
		ParentReviewResults: execution.ParentReviewTaskResults{
			"child-a": {
				TaskID: "child-a",
				Status: execution.ParentReviewTaskStatusPassed,
			},
			"child-b": {
				TaskID: "child-b",
				Status: execution.ParentReviewTaskStatusPassed,
			},
		},
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-checkout",
				Title: "Parent Checkout Review",
			},
		},
	}
}

func testParentReviewAllFailedRun() execution.RunRecord {
	passed := false
	return execution.RunRecord{
		ID:                 "review-88",
		TaskID:             "parent-checkout",
		ParentReviewPassed: &passed,
		ParentReviewResults: execution.ParentReviewTaskResults{
			"child-a": {
				TaskID:   "child-a",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Fix child-a issues.",
			},
			"child-b": {
				TaskID:   "child-b",
				Status:   execution.ParentReviewTaskStatusFailed,
				Feedback: "Fix child-b issues.",
			},
		},
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "parent-checkout",
				Title: "Parent Checkout Review",
			},
		},
	}
}
