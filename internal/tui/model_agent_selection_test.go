package tui

import (
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestAgentSelectionSavedUpdatesModel(t *testing.T) {
	model := Model{
		agentSelection: agent.AgentSelection{
			Agent:         agent.AgentInfo{ID: agent.AgentClaude, Label: "Claude"},
			ConfigPresent: true,
		},
		agentSelectionErr: "previous error",
	}

	msg := AgentSelectionSaved{
		Selection: agent.AgentSelection{
			Agent:         agent.AgentInfo{ID: agent.AgentCodex, Label: "Codex"},
			ConfigPresent: true,
		},
	}

	updated, _ := model.Update(msg)
	next := updated.(Model)

	if next.agentSelection.Agent.ID != agent.AgentCodex {
		t.Fatalf("expected agent selection to update, got %q", next.agentSelection.Agent.ID)
	}
	if next.agentSelectionErr != "" {
		t.Fatalf("expected agent selection error to clear, got %q", next.agentSelectionErr)
	}
	if next.actionOutput == nil || next.actionOutput.IsError {
		t.Fatalf("expected success action output, got %#v", next.actionOutput)
	}
	if !strings.Contains(next.actionOutput.Message, "Agent set to Codex") {
		t.Fatalf("expected action output to mention agent label, got %q", next.actionOutput.Message)
	}
}
