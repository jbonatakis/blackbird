package index

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

func TestScoreResultRecencyAndType(t *testing.T) {
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	opts := SearchOptions{
		Now:             now,
		RecencyHalfLife: 24 * time.Hour,
		TypeWeights: map[artifact.ArtifactType]float64{
			artifact.ArtifactDecision:   3.0,
			artifact.ArtifactTranscript: 1.0,
		},
	}

	newDecision := scoreResult(1.0, artifact.ArtifactDecision, now, opts)
	oldDecision := scoreResult(1.0, artifact.ArtifactDecision, now.Add(-24*time.Hour), opts)
	newTranscript := scoreResult(1.0, artifact.ArtifactTranscript, now, opts)

	if newDecision <= oldDecision {
		t.Fatalf("expected newer decision to score higher than older decision")
	}
	if oldDecision <= newTranscript {
		t.Fatalf("expected decision weighting to outrank transcript")
	}
}

func TestSearchFiltersAndSnippetBounds(t *testing.T) {
	longText := strings.Repeat("index artifacts for fast search ", 10)
	artifacts := []artifact.Artifact{
		newArtifact("a1", "s1", "t1", "r1", artifact.ArtifactDecision, "Decision: use sqlite index for memory."),
		newArtifact("a2", "s2", "t1", "r2", artifact.ArtifactTranscript, longText),
		newArtifact("a3", "s1", "t2", "r3", artifact.ArtifactOutcome, "Outcome: retrieval API done."),
	}

	idx := buildIndex(t, artifacts, map[string]time.Time{
		"a1": time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		"a2": time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC),
		"a3": time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
	})

	results, err := idx.Search(SearchOptions{
		Query:         "index",
		Types:         []artifact.ArtifactType{artifact.ArtifactDecision},
		SnippetMaxLen: 40,
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ArtifactID != "a1" {
		t.Fatalf("expected decision artifact a1, got %s", results[0].ArtifactID)
	}
	if len([]rune(results[0].Snippet)) > 40 {
		t.Fatalf("snippet length = %d, want <= 40", len([]rune(results[0].Snippet)))
	}

	sessionResults, err := idx.Search(SearchOptions{
		Query:     "index",
		SessionID: "s2",
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("search by session: %v", err)
	}
	if len(sessionResults) != 1 || sessionResults[0].ArtifactID != "a2" {
		t.Fatalf("expected session filtered result a2, got %+v", sessionResults)
	}
}
