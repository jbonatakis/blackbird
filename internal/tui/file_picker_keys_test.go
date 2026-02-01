package tui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleFilePickerKeyOpenOnAt(t *testing.T) {
	state := NewFilePickerState()
	queries := []string{}
	matcher := stubFilePickerMatcher(&queries, map[string][]string{
		"": {"a.txt", "b.txt"},
	})

	opts := FilePickerKeyOptions{
		Field:   "description",
		Anchor:  FilePickerAnchor{Start: 3},
		Matcher: matcher,
	}

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionNone {
		t.Fatalf("expected action none, got %v", result.Action)
	}
	if !state.Open {
		t.Fatalf("expected picker to be open")
	}
	if state.ActiveField != "description" {
		t.Fatalf("expected active field to be set")
	}
	if state.Anchor.Start != 3 {
		t.Fatalf("expected anchor start=3, got %d", state.Anchor.Start)
	}
	if state.Query != "" {
		t.Fatalf("expected empty query, got %q", state.Query)
	}
	if !reflect.DeepEqual(state.Matches, []string{"a.txt", "b.txt"}) {
		t.Fatalf("expected matches to be seeded, got %v", state.Matches)
	}
	if state.Selected != 0 {
		t.Fatalf("expected selected=0, got %d", state.Selected)
	}
	if !reflect.DeepEqual(queries, []string{""}) {
		t.Fatalf("expected matcher to be called with empty query, got %v", queries)
	}
}

func TestHandleFilePickerKeyMovesSelection(t *testing.T) {
	state := FilePickerState{Open: true, Matches: []string{"a", "b", "c"}, Selected: 1}

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyUp}, FilePickerKeyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Selected != 0 {
		t.Fatalf("expected selected=0, got %d", state.Selected)
	}
	if !result.Consumed {
		t.Fatalf("expected key to be consumed")
	}

	result, err = HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyDown}, FilePickerKeyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Selected != 1 {
		t.Fatalf("expected selected=1, got %d", state.Selected)
	}
	if !result.Consumed {
		t.Fatalf("expected key to be consumed")
	}
}

func TestHandleFilePickerKeyEnterSelects(t *testing.T) {
	state := FilePickerState{Open: true, Matches: []string{"a", "b"}, Selected: 1}

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyEnter}, FilePickerKeyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionInsert {
		t.Fatalf("expected insert action, got %v", result.Action)
	}
	if result.Selected != "b" {
		t.Fatalf("expected selected match 'b', got %q", result.Selected)
	}
	if state.Open {
		t.Fatalf("expected picker to close after insert")
	}
	if !result.Consumed {
		t.Fatalf("expected key to be consumed")
	}
}

func TestHandleFilePickerKeyEscCancels(t *testing.T) {
	state := FilePickerState{Open: true, Matches: []string{"a"}, Selected: 0}

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyEsc}, FilePickerKeyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionCancel {
		t.Fatalf("expected cancel action, got %v", result.Action)
	}
	if state.Open {
		t.Fatalf("expected picker to close on cancel")
	}
	if !result.Consumed {
		t.Fatalf("expected key to be consumed")
	}
}

func TestHandleFilePickerKeyTabCancelsWithoutConsuming(t *testing.T) {
	state := FilePickerState{Open: true, Matches: []string{"a"}, Selected: 0}

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyTab}, FilePickerKeyOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionCancel {
		t.Fatalf("expected cancel action, got %v", result.Action)
	}
	if state.Open {
		t.Fatalf("expected picker to close on tab")
	}
	if result.Consumed {
		t.Fatalf("expected tab to pass through")
	}
}

func TestHandleFilePickerKeyBackspaceUpdatesQuery(t *testing.T) {
	state := FilePickerState{Open: true, Query: "src/", Matches: []string{"src/a"}, Selected: 0}
	queries := []string{}
	matcher := stubFilePickerMatcher(&queries, map[string][]string{
		"src": {"src/b"},
	})

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyBackspace}, FilePickerKeyOptions{Matcher: matcher})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionNone {
		t.Fatalf("expected action none, got %v", result.Action)
	}
	if state.Query != "src" {
		t.Fatalf("expected query 'src', got %q", state.Query)
	}
	if !reflect.DeepEqual(state.Matches, []string{"src/b"}) {
		t.Fatalf("expected matches to update, got %v", state.Matches)
	}
	if !reflect.DeepEqual(queries, []string{"src"}) {
		t.Fatalf("expected matcher to be called with 'src', got %v", queries)
	}
}

func TestHandleFilePickerKeyPrintableExtendsQuery(t *testing.T) {
	state := FilePickerState{Open: true, Query: "src/", Matches: []string{"src/a"}, Selected: 0}
	queries := []string{}
	matcher := stubFilePickerMatcher(&queries, map[string][]string{
		"src/m": {"src/main.go"},
	})

	result, err := HandleFilePickerKey(&state, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}, FilePickerKeyOptions{Matcher: matcher})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != FilePickerActionNone {
		t.Fatalf("expected action none, got %v", result.Action)
	}
	if state.Query != "src/m" {
		t.Fatalf("expected query 'src/m', got %q", state.Query)
	}
	if !reflect.DeepEqual(state.Matches, []string{"src/main.go"}) {
		t.Fatalf("expected matches to update, got %v", state.Matches)
	}
	if !reflect.DeepEqual(queries, []string{"src/m"}) {
		t.Fatalf("expected matcher to be called with 'src/m', got %v", queries)
	}
}

func stubFilePickerMatcher(queries *[]string, responses map[string][]string) FilePickerMatcher {
	return func(query string, limit int) ([]string, error) {
		*queries = append(*queries, query)
		if matches, ok := responses[query]; ok {
			return matches, nil
		}
		return nil, nil
	}
}
