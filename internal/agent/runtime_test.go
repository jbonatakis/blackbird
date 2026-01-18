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

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
