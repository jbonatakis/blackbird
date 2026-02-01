package tui

import "unicode/utf8"

// FilePickerField identifies which input field owns the active file picker.
type FilePickerField string

const FilePickerFieldNone FilePickerField = ""

// FilePickerAnchor stores the position of the "@" trigger for replacement.
// Start is a rune offset into the field value; Line/Column are optional helpers.
type FilePickerAnchor struct {
	Start  int
	Line   int
	Column int
}

// FilePickerState tracks the open/closed state and selection of the @ file picker.
type FilePickerState struct {
	Open        bool
	Query       string
	Matches     []string
	Selected    int
	ActiveField FilePickerField
	Anchor      FilePickerAnchor
}

func NewFilePickerState() FilePickerState {
	return FilePickerState{
		Selected: -1,
	}
}

func (s *FilePickerState) Reset() {
	*s = NewFilePickerState()
}

func (s *FilePickerState) OpenAt(field FilePickerField, anchor FilePickerAnchor) {
	s.Reset()
	s.Open = true
	s.ActiveField = field
	s.Anchor = anchor
}

func (s *FilePickerState) Close() {
	s.Open = false
	s.Query = ""
	s.Matches = nil
	s.Selected = -1
}

func (s *FilePickerState) SetMatches(matches []string) {
	s.Matches = matches
	s.ClampSelection()
}

func (s *FilePickerState) ClampSelection() {
	if len(s.Matches) == 0 {
		s.Selected = -1
		return
	}
	if s.Selected < 0 {
		s.Selected = 0
		return
	}
	if s.Selected >= len(s.Matches) {
		s.Selected = len(s.Matches) - 1
	}
}

func (s *FilePickerState) MoveSelection(delta int) {
	if len(s.Matches) == 0 {
		s.Selected = -1
		return
	}
	if s.Selected < 0 {
		s.Selected = 0
	}
	s.Selected += delta
	if s.Selected < 0 {
		s.Selected = 0
		return
	}
	if s.Selected >= len(s.Matches) {
		s.Selected = len(s.Matches) - 1
	}
}

func (s FilePickerState) HasSelection() bool {
	return s.Selected >= 0 && s.Selected < len(s.Matches)
}

func (s FilePickerState) SelectedMatch() (string, bool) {
	if !s.HasSelection() {
		return "", false
	}
	return s.Matches[s.Selected], true
}

// AnchorSpan returns the rune range covering the @ + current query text.
func (s FilePickerState) AnchorSpan() (start, end int) {
	start = s.Anchor.Start
	if start < 0 {
		start = 0
	}
	end = start + 1 + utf8.RuneCountInString(s.Query)
	return start, end
}
