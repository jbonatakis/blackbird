package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/execution"
)

func TestPromptReviewDecisionLineSelectsByNumber(t *testing.T) {
	options := defaultReviewDecisionOptions()
	var selected reviewDecisionOption

	_, err := captureStdout(func() error {
		return withPromptInput("2\n", func() error {
			var innerErr error
			selected, innerErr = promptReviewDecisionLine(options)
			return innerErr
		})
	})
	if err != nil {
		t.Fatalf("promptReviewDecisionLine: %v", err)
	}
	if selected.Action != execution.DecisionStateApprovedQuit {
		t.Fatalf("expected approved quit, got %v", selected.Action)
	}
}

func TestPromptReviewFeedback_FilePickerAndRetry(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	if err := os.MkdirAll("src", 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile("src/main.go", []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile("src/other.go", []byte("package other\n"), 0o644); err != nil {
		t.Fatalf("write other.go: %v", err)
	}

	var feedback string
	output, err := captureStdout(func() error {
		return withPromptInput("\nPlease update @src/\n1\n\n", func() error {
			var innerErr error
			feedback, innerErr = promptReviewFeedback()
			return innerErr
		})
	})
	if err != nil {
		t.Fatalf("promptReviewFeedback: %v", err)
	}
	if feedback != "Please update @src/main.go" {
		t.Fatalf("expected file picker replacement, got %q", feedback)
	}
	if !strings.Contains(output, "change request cannot be empty") {
		t.Fatalf("expected empty feedback retry message, got %q", output)
	}
	if !strings.Contains(output, "File picker for @src/") {
		t.Fatalf("expected file picker prompt, got %q", output)
	}
}
