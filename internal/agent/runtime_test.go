package agent

import "testing"

func TestBuildFlagArgsJSONSchemaClaudeOnly(t *testing.T) {
	meta := RequestMetadata{
		JSONSchema: `{"type":"object"}`,
	}

	claudeArgs := buildFlagArgs("claude", meta)
	if !containsArg(claudeArgs, "--json-schema") {
		t.Fatalf("expected --json-schema for claude provider")
	}

	codexArgs := buildFlagArgs("codex", meta)
	if containsArg(codexArgs, "--json-schema") {
		t.Fatalf("did not expect --json-schema for codex provider")
	}
}

func TestBuildFlagArgsJSONSchemaEmpty(t *testing.T) {
	meta := RequestMetadata{}
	args := buildFlagArgs("claude", meta)
	if containsArg(args, "--json-schema") {
		t.Fatalf("did not expect --json-schema when JSONSchema is empty")
	}
}

func TestApplyProviderArgs(t *testing.T) {
	codex := applyProviderArgs("codex", []string{"--model", "foo"})
	if len(codex) < 2 || codex[0] != "exec" || codex[1] != "--full-auto" {
		t.Fatalf("expected codex exec --full-auto prefix, got %v", codex)
	}

	claude := applyProviderArgs("claude", []string{"--model", "foo"})
	if len(claude) < 2 || claude[0] != "--permission-mode" || claude[1] != "bypassPermissions" {
		t.Fatalf("expected claude permission-mode prefix, got %v", claude)
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
