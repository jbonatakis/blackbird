package agent

import (
	"reflect"
	"testing"
)

func TestSupportedAgentIDs(t *testing.T) {
	expected := []AgentID{AgentClaude, AgentCodex}
	if got := SupportedAgentIDs(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("SupportedAgentIDs() = %#v, want %#v", got, expected)
	}
}

func TestLookupAgent(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantID  AgentID
		wantOK  bool
		wantLbl string
	}{
		{name: "claude lower", id: "claude", wantID: AgentClaude, wantLbl: "Claude", wantOK: true},
		{name: "claude mixed", id: " Claude ", wantID: AgentClaude, wantLbl: "Claude", wantOK: true},
		{name: "codex lower", id: "codex", wantID: AgentCodex, wantLbl: "Codex", wantOK: true},
		{name: "codex mixed", id: " CODEX ", wantID: AgentCodex, wantLbl: "Codex", wantOK: true},
		{name: "unknown", id: "unknown", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LookupAgent(tc.id)
			if ok != tc.wantOK {
				t.Fatalf("LookupAgent(%q) ok=%v, want %v", tc.id, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got.ID != tc.wantID {
				t.Fatalf("LookupAgent(%q) ID=%q, want %q", tc.id, got.ID, tc.wantID)
			}
			if got.Label != tc.wantLbl {
				t.Fatalf("LookupAgent(%q) Label=%q, want %q", tc.id, got.Label, tc.wantLbl)
			}
		})
	}
}

func TestDefaultAgent(t *testing.T) {
	got := DefaultAgent()
	if got.ID != AgentClaude {
		t.Fatalf("DefaultAgent() ID=%q, want %q", got.ID, AgentClaude)
	}
	if got.Label == "" {
		t.Fatalf("DefaultAgent() Label should not be empty")
	}
}
