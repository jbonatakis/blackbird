package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const AgentSelectionSchemaVersion = 1

type AgentSelection struct {
	Agent         AgentInfo
	ConfigPresent bool
}

type agentSelectionFile struct {
	SchemaVersion int    `json:"schemaVersion"`
	SelectedAgent string `json:"selectedAgent"`
}

func AgentSelectionPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Join(".blackbird", "agent.json")
	}
	return filepath.Join(wd, ".blackbird", "agent.json")
}

func LoadAgentSelection(path string) (AgentSelection, error) {
	defaultSelection := func(present bool) AgentSelection {
		return AgentSelection{
			Agent:         DefaultAgent(),
			ConfigPresent: present,
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultSelection(false), nil
		}
		return defaultSelection(false), fmt.Errorf("read agent selection %s: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	var cfg agentSelectionFile
	if err := dec.Decode(&cfg); err != nil {
		return defaultSelection(true), fmt.Errorf("parse agent selection %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return defaultSelection(true), fmt.Errorf("parse agent selection %s: trailing JSON values", path)
		}
		return defaultSelection(true), fmt.Errorf("parse agent selection %s: trailing data: %w", path, err)
	}

	if cfg.SchemaVersion != AgentSelectionSchemaVersion {
		return defaultSelection(true), fmt.Errorf("unsupported agent selection schema version %d", cfg.SchemaVersion)
	}
	if cfg.SelectedAgent == "" {
		return defaultSelection(true), errors.New("agent selection is missing selectedAgent")
	}

	info, ok := LookupAgent(cfg.SelectedAgent)
	if !ok {
		return defaultSelection(true), fmt.Errorf("unsupported agent selection %q", cfg.SelectedAgent)
	}

	return AgentSelection{
		Agent:         info,
		ConfigPresent: true,
	}, nil
}

func SaveAgentSelection(path string, selectedAgent string) error {
	info, ok := LookupAgent(selectedAgent)
	if !ok {
		return fmt.Errorf("unsupported agent selection %q", selectedAgent)
	}

	cfg := agentSelectionFile{
		SchemaVersion: AgentSelectionSchemaVersion,
		SelectedAgent: string(info.ID),
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode agent selection: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create agent selection dir: %w", err)
	}

	if err := atomicWriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write agent selection %s: %w", path, err)
	}
	return nil
}
