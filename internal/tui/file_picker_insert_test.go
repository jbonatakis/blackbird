package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReplaceFilePickerSpanSingleLine(t *testing.T) {
	value := "Use @src/ for details"
	anchor := FilePickerAnchor{Start: runeIndex(value, "@")}
	query := "src/"

	updated, cursor := replaceFilePickerSpan(value, anchor, query, "src/main.go")

	expected := "Use @src/main.go for details"
	if updated != expected {
		t.Fatalf("expected %q, got %q", expected, updated)
	}
	expectedCursor := anchor.Start + utf8.RuneCountInString("@src/main.go")
	if cursor != expectedCursor {
		t.Fatalf("expected cursor %d, got %d", expectedCursor, cursor)
	}
}

func TestReplaceFilePickerSpanMultiline(t *testing.T) {
	value := "Line1\nSee @internal/ for details\nLine3"
	anchor := FilePickerAnchor{Start: runeIndex(value, "@")}
	query := "internal/"

	updated, cursor := replaceFilePickerSpan(value, anchor, query, "internal/tui/model.go")

	expected := "Line1\nSee @internal/tui/model.go for details\nLine3"
	if updated != expected {
		t.Fatalf("expected %q, got %q", expected, updated)
	}
	expectedCursor := anchor.Start + utf8.RuneCountInString("@internal/tui/model.go")
	if cursor != expectedCursor {
		t.Fatalf("expected cursor %d, got %d", expectedCursor, cursor)
	}
}

func runeIndex(s string, substr string) int {
	idx := strings.Index(s, substr)
	if idx < 0 {
		return -1
	}
	return utf8.RuneCountInString(s[:idx])
}
