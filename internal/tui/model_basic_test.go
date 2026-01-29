package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateQuitCommand(t *testing.T) {
	model := Model{}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command to return tea.QuitMsg")
	}
}

func TestWindowSizeMsgUpdatesDimensions(t *testing.T) {
	model := Model{}
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	updated, _ := model.Update(msg)
	updatedModel := updated.(Model)

	if updatedModel.windowWidth != 120 {
		t.Fatalf("expected width 120, got %d", updatedModel.windowWidth)
	}
	if updatedModel.windowHeight != 40 {
		t.Fatalf("expected height 40, got %d", updatedModel.windowHeight)
	}
}

func TestViewRendersPlaceholderText(t *testing.T) {
	model := Model{windowHeight: 2}

	view := model.View()
	if !strings.Contains(view, "No items.") {
		t.Fatalf("expected 'No items.' text in view for empty plan, got %q", view)
	}
}
