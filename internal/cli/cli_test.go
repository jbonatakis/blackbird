package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// TestRunZeroArgsLaunchesTUI verifies that calling Run with no arguments
// attempts to launch the TUI (tui.Start()). Since we can't fully mock the
// Bubble Tea program without significant refactoring, we verify that:
// 1. With a valid plan file, it attempts to start (would hang in real terminal)
// 2. Without a plan file, it returns the expected error message
func TestRunZeroArgsWithoutPlanFile(t *testing.T) {
	// Set up temporary directory without a plan file
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

	// Run with no args should try to start TUI and fail with plan not found
	err = Run([]string{})
	if err == nil {
		t.Fatal("expected error when plan file not found, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "plan file not found") {
		t.Errorf("expected 'plan file not found' error, got: %v", err)
	}
	if !strings.Contains(errMsg, "blackbird init") {
		t.Errorf("expected error to suggest 'blackbird init', got: %v", err)
	}
}

// TestRunZeroArgsWithValidPlan verifies that with a valid plan file,
// Run([]) would attempt to launch the TUI. We can't test the full TUI
// without a TTY, but we can verify the setup doesn't error immediately.
func TestRunZeroArgsWithValidPlan(t *testing.T) {
	t.Skip("Skipping full TUI launch test - requires TTY and would hang")

	// This test is kept as documentation of the expected behavior:
	// When Run([]) is called with a valid plan file in the working directory,
	// it should call tui.Start() which launches the Bubble Tea program.
	//
	// In a real terminal, this would:
	// 1. Load the plan file
	// 2. Initialize the TUI model
	// 3. Start the Bubble Tea program
	// 4. Display the interactive TUI
	//
	// The test is skipped because:
	// - tui.Start() requires a TTY (terminal) to run
	// - It would hang waiting for user input
	// - Full TUI testing is better done manually or with specialized tools
}

// TestRunHelpCommand verifies the help command works
func TestRunHelpCommand(t *testing.T) {
	// Capture stdout
	output, err := captureStdout(func() error {
		return Run([]string{"help"})
	})

	if err != nil {
		t.Fatalf("Run(help) returned error: %v", err)
	}

	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output to contain 'Usage:', got: %q", output)
	}
	if !strings.Contains(output, "blackbird") {
		t.Errorf("expected help output to mention 'blackbird', got: %q", output)
	}
}

// TestRunInitCommand verifies the init command creates a plan file
func TestRunInitCommand(t *testing.T) {
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

	// Run init
	err = Run([]string{"init"})
	if err != nil {
		t.Fatalf("Run(init) returned error: %v", err)
	}

	// Verify plan file was created
	planFile := planPath()
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Errorf("plan file was not created at: %s", planFile)
	}

	// Verify plan file is valid
	g, err := plan.Load(planFile)
	if err != nil {
		t.Errorf("created plan file is not valid: %v", err)
	}
	if g.SchemaVersion != plan.SchemaVersion {
		t.Errorf("plan schema version = %d, want %d", g.SchemaVersion, plan.SchemaVersion)
	}
}

// TestRunValidateCommand verifies the validate command works with valid plan
func TestRunValidateCommand(t *testing.T) {
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

	// Create a valid plan - must include all required fields
	now := time.Now().UTC()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:                 "task-1",
				Title:              "Test Task",
				Description:        "Test description",
				AcceptanceCriteria: []string{},
				Prompt:             "Test prompt",
				ParentID:           nil,
				Status:             plan.StatusTodo,
				ChildIDs:           []string{},
				Deps:               []string{},
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(planPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	// Run validate
	output, err := captureStdout(func() error {
		return Run([]string{"validate"})
	})
	if err != nil {
		t.Fatalf("Run(validate) returned error: %v", err)
	}

	if !strings.Contains(output, "OK") {
		t.Errorf("expected validation success message (containing 'OK'), got: %q", output)
	}
}
