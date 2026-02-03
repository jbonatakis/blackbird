package artifact

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/memory/canonical"
)

func TestExtractArtifactsTypes(t *testing.T) {
	log := canonical.Log{
		SessionID: "sess-1",
		TaskID:    "task-1",
		RunID:     "run-1",
		Items: []canonical.Item{
			{
				Type: canonical.ItemMessage,
				Message: &canonical.Message{
					Role:    "user",
					Content: "Please summarize the plan.",
				},
			},
			{
				Type: canonical.ItemMessage,
				Message: &canonical.Message{
					Role: "assistant",
					Content: "Decision: Use JSONL store.\n" +
						"Constraint: Must keep deterministic output.\n" +
						"TODO: Add dedup rules.\n" +
						"All set.",
				},
			},
		},
	}

	artifacts := BuildArtifacts(nil, []canonical.Log{log})
	counts := map[ArtifactType]int{}
	for _, art := range artifacts {
		counts[art.Type]++
		if art.BuilderVersion == "" {
			t.Fatalf("artifact %s missing builder version", art.ArtifactID)
		}
	}

	if counts[ArtifactTranscript] != 2 {
		t.Fatalf("transcript artifacts = %d, want 2", counts[ArtifactTranscript])
	}
	if counts[ArtifactDecision] != 1 {
		t.Fatalf("decision artifacts = %d, want 1", counts[ArtifactDecision])
	}
	if counts[ArtifactConstraint] != 1 {
		t.Fatalf("constraint artifacts = %d, want 1", counts[ArtifactConstraint])
	}
	if counts[ArtifactOpenThread] != 1 {
		t.Fatalf("open thread artifacts = %d, want 1", counts[ArtifactOpenThread])
	}
	if counts[ArtifactOutcome] != 1 {
		t.Fatalf("outcome artifacts = %d, want 1", counts[ArtifactOutcome])
	}
}

func TestDedupDecisions(t *testing.T) {
	log := canonical.Log{
		RunID: "run-1",
		Items: []canonical.Item{
			{Type: canonical.ItemMessage, Message: &canonical.Message{Role: "assistant", Content: "Decision: Store uses JSONL."}},
			{Type: canonical.ItemMessage, Message: &canonical.Message{Role: "assistant", Content: "Decision: Store uses JSONL."}},
		},
	}

	artifacts := BuildArtifacts(nil, []canonical.Log{log})
	decisions := 0
	for _, art := range artifacts {
		if art.Type == ArtifactDecision {
			decisions++
		}
	}
	if decisions != 1 {
		t.Fatalf("decision artifacts = %d, want 1", decisions)
	}
}

func TestSupersedesDecision(t *testing.T) {
	log := canonical.Log{
		RunID: "run-1",
		Items: []canonical.Item{
			{Type: canonical.ItemMessage, Message: &canonical.Message{Role: "assistant", Content: "Decision: Storage uses JSONL for artifacts."}},
			{Type: canonical.ItemMessage, Message: &canonical.Message{Role: "assistant", Content: "Decision: Storage uses SQLite for artifacts."}},
		},
	}

	artifacts := BuildArtifacts(nil, []canonical.Log{log})
	var first, second *Artifact
	for i := range artifacts {
		art := &artifacts[i]
		if art.Type != ArtifactDecision {
			continue
		}
		switch art.Content.Text {
		case "Storage uses JSONL for artifacts.":
			first = art
		case "Storage uses SQLite for artifacts.":
			second = art
		}
	}
	if first == nil || second == nil {
		t.Fatalf("expected both decisions, got first=%v second=%v", first != nil, second != nil)
	}
	if !contains(second.Supersedes, first.ArtifactID) {
		t.Fatalf("second decision does not supersede first")
	}
	if !contains(first.SupersededBy, second.ArtifactID) {
		t.Fatalf("first decision not marked superseded by second")
	}
}
