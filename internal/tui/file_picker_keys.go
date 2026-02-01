package tui

import (
	"strconv"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// FilePickerAction describes the caller-visible outcome of handling a key.
type FilePickerAction int

const (
	FilePickerActionNone FilePickerAction = iota
	FilePickerActionInsert
	FilePickerActionCancel
)

// FilePickerKeyResult reports how the caller should respond to a key event.
type FilePickerKeyResult struct {
	Action   FilePickerAction
	Selected string
	Consumed bool
}

// FilePickerMatcher fetches matches for the current query.
type FilePickerMatcher func(query string, limit int) ([]string, error)

// FilePickerKeyOptions provide context for handling file picker keys.
type FilePickerKeyOptions struct {
	Field   FilePickerField
	Anchor  FilePickerAnchor
	Limit   int
	Matcher FilePickerMatcher
}

// HandleFilePickerKey routes key input to the file picker and returns the
// resulting action plus selection info when applicable.
func HandleFilePickerKey(state *FilePickerState, msg tea.KeyMsg, opts FilePickerKeyOptions) (FilePickerKeyResult, error) {
	result := FilePickerKeyResult{Action: FilePickerActionNone}
	if state == nil {
		return result, nil
	}

	matcher := opts.Matcher
	if matcher == nil {
		matcher = listWorkspaceFiles
	}

	if !state.Open {
		if isFilePickerOpenKey(msg) {
			state.OpenAt(opts.Field, opts.Anchor)
			if err := updateFilePickerMatches(state, matcher, opts.Limit); err != nil {
				return result, err
			}
		}
		return result, nil
	}

	switch msg.String() {
	case "up":
		state.MoveSelection(-1)
		result.Consumed = true
		return result, nil
	case "down":
		state.MoveSelection(1)
		result.Consumed = true
		return result, nil
	case "enter":
		if selected, ok := state.SelectedMatch(); ok {
			result.Action = FilePickerActionInsert
			result.Selected = selected
			state.Close()
		}
		result.Consumed = true
		return result, nil
	case "esc":
		state.Close()
		result.Action = FilePickerActionCancel
		result.Consumed = true
		return result, nil
	case "tab", "shift+tab":
		state.Close()
		result.Action = FilePickerActionCancel
		return result, nil
	case "backspace":
		if state.Query == "" {
			return result, nil
		}
		state.Query = dropLastRune(state.Query)
		if err := updateFilePickerMatches(state, matcher, opts.Limit); err != nil {
			return result, err
		}
		return result, nil
	}

	if msg.Type == tea.KeyRunes {
		printable := printableRunes(msg.Runes)
		if len(printable) > 0 {
			state.Query += string(printable)
			if err := updateFilePickerMatches(state, matcher, opts.Limit); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

func isFilePickerOpenKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '@'
}

func updateFilePickerMatches(state *FilePickerState, matcher FilePickerMatcher, limit int) error {
	matches, err := matcher(state.Query, limit)
	if err != nil {
		return err
	}
	state.SetMatches(matches)
	return nil
}

func dropLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	if size <= 0 {
		return ""
	}
	return s[:len(s)-size]
}

func printableRunes(runes []rune) []rune {
	if len(runes) == 0 {
		return nil
	}
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if strconv.IsPrint(r) {
			out = append(out, r)
		}
	}
	return out
}
