package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderFilePickerListClosed(t *testing.T) {
	state := FilePickerState{}
	if got := RenderFilePickerList(state, 20, 3); got != "" {
		t.Fatalf("expected empty output when picker is closed, got %q", got)
	}
}

func TestRenderFilePickerListEmptyState(t *testing.T) {
	state := FilePickerState{Open: true}
	width := 40
	height := 3
	output := RenderFilePickerList(state, width, height)
	if !strings.Contains(output, filePickerEmptyMessage) {
		t.Fatalf("expected empty state message, got %q", output)
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d", height, len(lines))
	}
	for i, line := range lines {
		if lipgloss.Width(line) != width {
			t.Fatalf("expected line %d width %d, got %d", i, width, lipgloss.Width(line))
		}
	}
}

func TestRenderFilePickerListSelectionWindow(t *testing.T) {
	state := FilePickerState{
		Open:     true,
		Matches:  []string{"alpha", "bravo", "charlie", "delta"},
		Selected: 2,
	}
	output := RenderFilePickerList(state, 20, 2)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "bravo") || strings.Contains(lines[0], "> ") {
		t.Fatalf("expected first line to be unselected 'bravo', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "charlie") || !strings.Contains(lines[1], "> ") {
		t.Fatalf("expected second line to be selected 'charlie', got %q", lines[1])
	}
	if strings.Contains(output, "alpha") || strings.Contains(output, "delta") {
		t.Fatalf("expected window to exclude alpha/delta, got %q", output)
	}
}
