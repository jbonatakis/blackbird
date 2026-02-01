package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentSelectionMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	got, err := LoadAgentSelection(path)
	if err != nil {
		t.Fatalf("LoadAgentSelection() error = %v", err)
	}
	if got.ConfigPresent {
		t.Fatalf("LoadAgentSelection() ConfigPresent=true, want false")
	}
	if got.Agent.ID != DefaultAgent().ID {
		t.Fatalf("LoadAgentSelection() Agent=%q, want default %q", got.Agent.ID, DefaultAgent().ID)
	}
}

func TestLoadAgentSelectionValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"selectedAgent":"codex"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := LoadAgentSelection(path)
	if err != nil {
		t.Fatalf("LoadAgentSelection() error = %v", err)
	}
	if !got.ConfigPresent {
		t.Fatalf("LoadAgentSelection() ConfigPresent=false, want true")
	}
	if got.Agent.ID != AgentCodex {
		t.Fatalf("LoadAgentSelection() Agent=%q, want %q", got.Agent.ID, AgentCodex)
	}
}

func TestLoadAgentSelectionInvalidAgent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"selectedAgent":"nope"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := LoadAgentSelection(path)
	if err == nil {
		t.Fatalf("LoadAgentSelection() error = nil, want error")
	}
	if !got.ConfigPresent {
		t.Fatalf("LoadAgentSelection() ConfigPresent=false, want true")
	}
	if got.Agent.ID != DefaultAgent().ID {
		t.Fatalf("LoadAgentSelection() Agent=%q, want default %q", got.Agent.ID, DefaultAgent().ID)
	}
}

func TestLoadAgentSelectionInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"selectedAgent":`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := LoadAgentSelection(path)
	if err == nil {
		t.Fatalf("LoadAgentSelection() error = nil, want error")
	}
	if !got.ConfigPresent {
		t.Fatalf("LoadAgentSelection() ConfigPresent=false, want true")
	}
	if got.Agent.ID != DefaultAgent().ID {
		t.Fatalf("LoadAgentSelection() Agent=%q, want default %q", got.Agent.ID, DefaultAgent().ID)
	}
}

func TestLoadAgentSelectionInvalidConfigFallback(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "missing selectedAgent",
			data: `{"schemaVersion":1}`,
		},
		{
			name: "unsupported schema",
			data: `{"schemaVersion":99,"selectedAgent":"codex"}`,
		},
		{
			name: "trailing data",
			data: `{"schemaVersion":1,"selectedAgent":"codex"}{"schemaVersion":1,"selectedAgent":"claude"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "agent.json")
			if err := os.WriteFile(path, []byte(test.data), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			got, err := LoadAgentSelection(path)
			if err == nil {
				t.Fatalf("LoadAgentSelection() error = nil, want error")
			}
			if !got.ConfigPresent {
				t.Fatalf("LoadAgentSelection() ConfigPresent=false, want true")
			}
			if got.Agent.ID != DefaultAgent().ID {
				t.Fatalf("LoadAgentSelection() Agent=%q, want default %q", got.Agent.ID, DefaultAgent().ID)
			}
		})
	}
}

func TestSaveAgentSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config", "agent.json")

	if err := SaveAgentSelection(path, "codex"); err != nil {
		t.Fatalf("SaveAgentSelection() error = %v", err)
	}

	got, err := LoadAgentSelection(path)
	if err != nil {
		t.Fatalf("LoadAgentSelection() error = %v", err)
	}
	if !got.ConfigPresent {
		t.Fatalf("LoadAgentSelection() ConfigPresent=false, want true")
	}
	if got.Agent.ID != AgentCodex {
		t.Fatalf("LoadAgentSelection() Agent=%q, want %q", got.Agent.ID, AgentCodex)
	}
}

func TestSaveAgentSelectionInvalidAgent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	if err := SaveAgentSelection(path, "nope"); err == nil {
		t.Fatalf("SaveAgentSelection() error = nil, want error")
	}
}
