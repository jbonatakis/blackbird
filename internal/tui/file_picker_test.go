package tui

import (
	"path/filepath"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilePickerListingTable(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "internal", "tui", "model.go"))
	writeTestFile(t, filepath.Join(root, "internal", "tui", "file_picker.go"))
	writeTestFile(t, filepath.Join(root, "internal", "cli", "main.go"))
	writeTestFile(t, filepath.Join(root, "docs", "spec.md"))
	writeTestFile(t, filepath.Join(root, "README.md"))

	cases := []struct {
		name  string
		query string
		limit int
		want  []string
	}{
		{
			name:  "all files",
			query: "",
			limit: 10,
			want: []string{
				"README.md",
				"docs/spec.md",
				"internal/cli/main.go",
				"internal/tui/file_picker.go",
				"internal/tui/model.go",
			},
		},
		{
			name:  "internal prefix",
			query: "internal/",
			limit: 10,
			want: []string{
				"internal/cli/main.go",
				"internal/tui/file_picker.go",
				"internal/tui/model.go",
			},
		},
		{
			name:  "nested prefix",
			query: "internal/tui/",
			limit: 10,
			want: []string{
				"internal/tui/file_picker.go",
				"internal/tui/model.go",
			},
		},
		{
			name:  "empty matches",
			query: "missing/",
			limit: 10,
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches, err := listWorkspaceFilesFromRoot(root, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(matches, tc.want) {
				t.Fatalf("expected matches %v, got %v", tc.want, matches)
			}
		})
	}
}

func TestFilePickerFilteringTable(t *testing.T) {
	cases := []struct {
		name  string
		query string
		files []string
		limit int
		want  []string
	}{
		{
			name:  "prefix ordering",
			query: "internal/",
			files: []string{
				"internal/tui/z.go",
				"docs/readme.md",
				"internal/tui/a.go",
			},
			limit: 10,
			want: []string{
				"internal/tui/a.go",
				"internal/tui/z.go",
			},
		},
		{
			name:  "limit applied",
			query: "",
			files: []string{
				"c.go",
				"a.go",
				"b.go",
			},
			limit: 2,
			want: []string{
				"a.go",
				"b.go",
			},
		},
		{
			name:  "empty matches",
			query: "src/",
			files: []string{
				"a.go",
				"b.go",
			},
			limit: 10,
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches := filterFilePickerMatches(tc.query, tc.files, tc.limit)
			if !reflect.DeepEqual(matches, tc.want) {
				t.Fatalf("expected matches %v, got %v", tc.want, matches)
			}
		})
	}
}

func TestHandleFilePickerKeyActionsTable(t *testing.T) {
	cases := []struct {
		name               string
		state              FilePickerState
		msg                tea.KeyMsg
		opts               FilePickerKeyOptions
		matcherResponses   map[string][]string
		wantAction         FilePickerAction
		wantSelected       string
		wantConsumed       bool
		wantOpen           bool
		wantQuery          string
		wantMatches        []string
		wantSelectedIndex  int
		wantField          FilePickerField
		wantAnchorStart    int
		wantMatcherQueries []string
	}{
		{
			name:  "open on @",
			state: NewFilePickerState(),
			msg:   tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}},
			opts: FilePickerKeyOptions{
				Field:  "prompt",
				Anchor: FilePickerAnchor{Start: 5},
				Limit:  10,
			},
			matcherResponses: map[string][]string{
				"": {"a.txt", "b.txt"},
			},
			wantAction:         FilePickerActionNone,
			wantConsumed:       false,
			wantOpen:           true,
			wantQuery:          "",
			wantMatches:        []string{"a.txt", "b.txt"},
			wantSelectedIndex:  0,
			wantField:          "prompt",
			wantAnchorStart:    5,
			wantMatcherQueries: []string{""},
		},
		{
			name:              "enter inserts",
			state:             FilePickerState{Open: true, Matches: []string{"a", "b"}, Selected: 1},
			msg:               tea.KeyMsg{Type: tea.KeyEnter},
			opts:              FilePickerKeyOptions{},
			wantAction:        FilePickerActionInsert,
			wantSelected:      "b",
			wantConsumed:      true,
			wantOpen:          false,
			wantQuery:         "",
			wantMatches:       nil,
			wantSelectedIndex: -1,
		},
		{
			name:              "esc cancels",
			state:             FilePickerState{Open: true, Matches: []string{"a"}, Selected: 0},
			msg:               tea.KeyMsg{Type: tea.KeyEsc},
			opts:              FilePickerKeyOptions{},
			wantAction:        FilePickerActionCancel,
			wantConsumed:      true,
			wantOpen:          false,
			wantQuery:         "",
			wantMatches:       nil,
			wantSelectedIndex: -1,
		},
		{
			name:              "tab cancels",
			state:             FilePickerState{Open: true, Matches: []string{"a"}, Selected: 0},
			msg:               tea.KeyMsg{Type: tea.KeyTab},
			opts:              FilePickerKeyOptions{},
			wantAction:        FilePickerActionCancel,
			wantConsumed:      false,
			wantOpen:          false,
			wantQuery:         "",
			wantMatches:       nil,
			wantSelectedIndex: -1,
		},
		{
			name:              "enter with empty matches",
			state:             FilePickerState{Open: true, Selected: -1},
			msg:               tea.KeyMsg{Type: tea.KeyEnter},
			opts:              FilePickerKeyOptions{},
			wantAction:        FilePickerActionNone,
			wantConsumed:      true,
			wantOpen:          true,
			wantQuery:         "",
			wantMatches:       nil,
			wantSelectedIndex: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.state
			opts := tc.opts
			var queries []string
			if tc.matcherResponses != nil {
				opts.Matcher = stubFilePickerMatcher(&queries, tc.matcherResponses)
			}

			result, err := HandleFilePickerKey(&state, tc.msg, opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Action != tc.wantAction {
				t.Fatalf("expected action %v, got %v", tc.wantAction, result.Action)
			}
			if result.Selected != tc.wantSelected {
				t.Fatalf("expected selected %q, got %q", tc.wantSelected, result.Selected)
			}
			if result.Consumed != tc.wantConsumed {
				t.Fatalf("expected consumed=%v, got %v", tc.wantConsumed, result.Consumed)
			}
			if state.Open != tc.wantOpen {
				t.Fatalf("expected open=%v, got %v", tc.wantOpen, state.Open)
			}
			if state.Query != tc.wantQuery {
				t.Fatalf("expected query %q, got %q", tc.wantQuery, state.Query)
			}
			if !reflect.DeepEqual(state.Matches, tc.wantMatches) {
				t.Fatalf("expected matches %v, got %v", tc.wantMatches, state.Matches)
			}
			if state.Selected != tc.wantSelectedIndex {
				t.Fatalf("expected selected index %d, got %d", tc.wantSelectedIndex, state.Selected)
			}
			if state.ActiveField != tc.wantField {
				t.Fatalf("expected active field %q, got %q", tc.wantField, state.ActiveField)
			}
			if state.Anchor.Start != tc.wantAnchorStart {
				t.Fatalf("expected anchor start %d, got %d", tc.wantAnchorStart, state.Anchor.Start)
			}
			if tc.matcherResponses != nil && !reflect.DeepEqual(queries, tc.wantMatcherQueries) {
				t.Fatalf("expected matcher queries %v, got %v", tc.wantMatcherQueries, queries)
			}
		})
	}
}

func TestHandleFilePickerKeySelectionBoundsTable(t *testing.T) {
	cases := []struct {
		name     string
		selected int
		msg      tea.KeyMsg
		want     int
	}{
		{
			name:     "up clamps at top",
			selected: 0,
			msg:      tea.KeyMsg{Type: tea.KeyUp},
			want:     0,
		},
		{
			name:     "down clamps at bottom",
			selected: 2,
			msg:      tea.KeyMsg{Type: tea.KeyDown},
			want:     2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := FilePickerState{
				Open:     true,
				Matches:  []string{"a", "b", "c"},
				Selected: tc.selected,
			}

			result, err := HandleFilePickerKey(&state, tc.msg, FilePickerKeyOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Action != FilePickerActionNone {
				t.Fatalf("expected action none, got %v", result.Action)
			}
			if !result.Consumed {
				t.Fatalf("expected key to be consumed")
			}
			if state.Selected != tc.want {
				t.Fatalf("expected selected=%d, got %d", tc.want, state.Selected)
			}
		})
	}
}
