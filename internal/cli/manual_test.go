package cli

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunDelete_ForceReportsDetachedIDs(t *testing.T) {
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
			"B": func() plan.WorkItem {
				it := newWorkItem("B", now)
				it.Deps = []string{"A"}
				return it
			}(),
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error {
		return runDelete("A", []string{"--force"})
	})
	if err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(output, "deleted 1 item(s)") {
		t.Fatalf("output missing delete count: %q", output)
	}
	if !strings.Contains(output, "detached deps from: B") {
		t.Fatalf("output missing detached IDs: %q", output)
	}
}

func newWorkItem(id string, now time.Time) plan.WorkItem {
	return plan.WorkItem{
		ID:                 id,
		Title:              id,
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
}

func captureStdout(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	runErr := fn()
	_ = w.Close()
	os.Stdout = old

	data, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(data), runErr
}
