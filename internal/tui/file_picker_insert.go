package tui

import (
	"strings"
	"unicode/utf8"
)

// replaceFilePickerSpan replaces the "@query" span with "@selectedPath" and
// returns the updated value plus the cursor rune index after the insert.
func replaceFilePickerSpan(value string, anchor FilePickerAnchor, query string, selectedPath string) (string, int) {
	valueRunes := []rune(value)

	start := anchor.Start
	if start < 0 {
		start = 0
	}
	if start > len(valueRunes) {
		start = len(valueRunes)
	}

	spanLen := 1 + utf8.RuneCountInString(query)
	end := start + spanLen
	if end < start {
		end = start
	}
	if end > len(valueRunes) {
		end = len(valueRunes)
	}

	insert := "@" + strings.TrimPrefix(selectedPath, "@")
	insertRunes := []rune(insert)

	out := make([]rune, 0, len(valueRunes)-(end-start)+len(insertRunes))
	out = append(out, valueRunes[:start]...)
	out = append(out, insertRunes...)
	out = append(out, valueRunes[end:]...)

	cursor := start + len(insertRunes)
	return string(out), cursor
}
