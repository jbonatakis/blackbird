package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunPick_MarkDone(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	now := time.Now().UTC()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"A": newWorkItem("A", now),
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error {
		return withPromptInput("1\ndone\n", func() error {
			return runPick([]string{})
		})
	})
	if err != nil {
		t.Fatalf("runPick: %v", err)
	}
	if !strings.Contains(output, "Ready tasks:") {
		t.Fatalf("output missing tasks header: %q", output)
	}
	if !strings.Contains(output, "updated A status to done") {
		t.Fatalf("output missing status update: %q", output)
	}

	g, err = plan.Load(planPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	it := g.Items["A"]
	if it.Status != plan.StatusDone {
		t.Fatalf("expected status done, got %s", it.Status)
	}
}

func TestRunPick_NoReadyTasksMessage(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	now := time.Now().UTC()
	done := newWorkItem("A", now)
	done.Status = plan.StatusDone
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"A": done,
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error {
		return runPick([]string{})
	})
	if err != nil {
		t.Fatalf("runPick: %v", err)
	}
	if !strings.Contains(output, "0 ready") {
		t.Fatalf("output missing ready summary: %q", output)
	}
	if !strings.Contains(output, "list --blocked") {
		t.Fatalf("output missing suggestion: %q", output)
	}
}

func withPromptInput(input string, fn func() error) error {
	old := promptReader
	setPromptReader(strings.NewReader(input))
	defer func() {
		promptReader = old
	}()
	return fn()
}
