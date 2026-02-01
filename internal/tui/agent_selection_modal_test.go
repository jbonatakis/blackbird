package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestRenderAgentSelectionModal(t *testing.T) {
	model := Model{
		actionMode: ActionModeSelectAgent,
		agentSelection: agent.AgentSelection{
			Agent:         agent.DefaultAgent(),
			ConfigPresent: false,
		},
	}

	out := RenderAgentSelectionModal(model)
	if !strings.Contains(out, "Select agent") {
		t.Fatalf("expected modal title, got %q", out)
	}
	if !strings.Contains(out, "Claude") || !strings.Contains(out, "Codex") {
		t.Fatalf("expected modal to list agents, got %q", out)
	}
}

func TestHandleAgentSelectionKeyEsc(t *testing.T) {
	model := Model{actionMode: ActionModeSelectAgent}
	updated, _ := HandleAgentSelectionKey(model, "esc")
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode to reset on esc")
	}
}

func TestHandleAgentSelectionKeyUpDown(t *testing.T) {
	model := Model{
		actionMode:              ActionModeSelectAgent,
		agentSelectionHighlight: 0,
	}
	// Down moves to 1
	updated, cmd := HandleAgentSelectionKey(model, "down")
	if updated.agentSelectionHighlight != 1 {
		t.Fatalf("expected highlight 1 after down, got %d", updated.agentSelectionHighlight)
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd on down, got %v", cmd)
	}
	// Down again wraps to 0 (only 2 agents)
	updated, cmd = HandleAgentSelectionKey(updated, "down")
	if updated.agentSelectionHighlight != 0 {
		t.Fatalf("expected highlight 0 after wrap, got %d", updated.agentSelectionHighlight)
	}
	// Up from 0 wraps to last
	updated, cmd = HandleAgentSelectionKey(updated, "up")
	if updated.agentSelectionHighlight != 1 {
		t.Fatalf("expected highlight 1 after up wrap, got %d", updated.agentSelectionHighlight)
	}
}

func TestHandleAgentSelectionKeyEnterSelectsHighlighted(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".blackbird"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	model := Model{
		actionMode:              ActionModeSelectAgent,
		agentSelectionHighlight: 1, // Codex
	}
	updated, cmd := HandleAgentSelectionKey(model, "enter")
	if updated.actionMode != ActionModeNone {
		t.Fatalf("expected action mode to reset after enter")
	}
	if cmd == nil {
		t.Fatalf("expected save cmd after enter")
	}
	msg := cmd()
	saved, ok := msg.(AgentSelectionSaved)
	if !ok {
		t.Fatalf("expected AgentSelectionSaved, got %T", msg)
	}
	if saved.Err != nil {
		t.Fatalf("expected no error, got %v", saved.Err)
	}
	if saved.Selection.Agent.ID != agent.AgentCodex {
		t.Fatalf("expected codex selection, got %q", saved.Selection.Agent.ID)
	}
}

func TestSaveAgentSelectionCmd(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".blackbird"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	msg := SaveAgentSelectionCmd("codex")()
	selectionMsg, ok := msg.(AgentSelectionSaved)
	if !ok {
		t.Fatalf("expected AgentSelectionSaved, got %T", msg)
	}
	if selectionMsg.Err != nil {
		t.Fatalf("expected no error, got %v", selectionMsg.Err)
	}
	if selectionMsg.Selection.Agent.ID != agent.AgentCodex {
		t.Fatalf("expected codex selection, got %q", selectionMsg.Selection.Agent.ID)
	}
}
