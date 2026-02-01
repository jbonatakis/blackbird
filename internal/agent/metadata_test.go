package agent

import "testing"

func TestApplyRuntimeProviderRespectsExplicitMeta(t *testing.T) {
	meta := RequestMetadata{Provider: "claude"}
	runtime := Runtime{Provider: "codex"}
	got := ApplyRuntimeProvider(meta, runtime)
	if got.Provider != "claude" {
		t.Fatalf("expected provider claude, got %q", got.Provider)
	}
}

func TestApplyRuntimeProviderUsesRuntime(t *testing.T) {
	meta := RequestMetadata{}
	runtime := Runtime{Provider: "codex"}
	got := ApplyRuntimeProvider(meta, runtime)
	if got.Provider != "codex" {
		t.Fatalf("expected provider codex, got %q", got.Provider)
	}
}

func TestApplyRuntimeProviderEmptyRuntimeNoChange(t *testing.T) {
	meta := RequestMetadata{}
	runtime := Runtime{}
	got := ApplyRuntimeProvider(meta, runtime)
	if got.Provider != "" {
		t.Fatalf("expected empty provider, got %q", got.Provider)
	}
}
