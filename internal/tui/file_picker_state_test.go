package tui

import "testing"

func TestFilePickerStateDefaults(t *testing.T) {
	state := NewFilePickerState()

	if state.Open {
		t.Fatalf("expected Open=false by default")
	}
	if state.Query != "" {
		t.Fatalf("expected empty query, got %q", state.Query)
	}
	if len(state.Matches) != 0 {
		t.Fatalf("expected no matches by default")
	}
	if state.Selected != -1 {
		t.Fatalf("expected Selected=-1 by default, got %d", state.Selected)
	}
	if state.ActiveField != FilePickerFieldNone {
		t.Fatalf("expected ActiveField to be none by default")
	}
}

func TestFilePickerStateOpenCloseReset(t *testing.T) {
	state := NewFilePickerState()
	anchor := FilePickerAnchor{Start: 4, Line: 1, Column: 2}

	state.OpenAt("description", anchor)
	if !state.Open {
		t.Fatalf("expected picker to be open after OpenAt")
	}
	if state.ActiveField != "description" {
		t.Fatalf("expected ActiveField to be set, got %q", state.ActiveField)
	}
	if state.Anchor != anchor {
		t.Fatalf("expected anchor to be set")
	}

	state.Query = "src/"
	state.SetMatches([]string{"src/main.go"})
	state.Selected = 0

	state.Close()
	if state.Open {
		t.Fatalf("expected picker to be closed after Close")
	}
	if state.Query != "" {
		t.Fatalf("expected query to be cleared on Close")
	}
	if len(state.Matches) != 0 {
		t.Fatalf("expected matches to be cleared on Close")
	}
	if state.Selected != -1 {
		t.Fatalf("expected Selected=-1 on Close, got %d", state.Selected)
	}

	state.Reset()
	if state.ActiveField != FilePickerFieldNone {
		t.Fatalf("expected ActiveField to be reset")
	}
	if state.Anchor != (FilePickerAnchor{}) {
		t.Fatalf("expected anchor to be reset")
	}
}

func TestFilePickerStateClampSelection(t *testing.T) {
	state := NewFilePickerState()

	state.Selected = 2
	state.SetMatches([]string{})
	if state.Selected != -1 {
		t.Fatalf("expected Selected=-1 for empty matches, got %d", state.Selected)
	}

	state.Selected = -5
	state.SetMatches([]string{"a", "b"})
	if state.Selected != 0 {
		t.Fatalf("expected Selected=0 when negative, got %d", state.Selected)
	}

	state.Selected = 5
	state.SetMatches([]string{"a", "b"})
	if state.Selected != 1 {
		t.Fatalf("expected Selected to clamp to last index, got %d", state.Selected)
	}
}

func TestFilePickerStateMoveSelection(t *testing.T) {
	state := NewFilePickerState()
	state.SetMatches([]string{"a", "b", "c"})
	state.MoveSelection(1)
	if state.Selected != 1 {
		t.Fatalf("expected Selected=1 after move, got %d", state.Selected)
	}
	state.MoveSelection(10)
	if state.Selected != 2 {
		t.Fatalf("expected Selected to clamp to end, got %d", state.Selected)
	}
	state.MoveSelection(-10)
	if state.Selected != 0 {
		t.Fatalf("expected Selected to clamp to start, got %d", state.Selected)
	}
}

func TestFilePickerAnchorSpan(t *testing.T) {
	state := NewFilePickerState()
	state.Anchor = FilePickerAnchor{Start: 3}
	state.Query = "src/"

	start, end := state.AnchorSpan()
	if start != 3 {
		t.Fatalf("expected start=3, got %d", start)
	}
	if end != 8 {
		t.Fatalf("expected end=8, got %d", end)
	}
}
