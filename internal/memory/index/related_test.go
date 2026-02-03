package index

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

func TestRelatedByRunTaskAndProvenance(t *testing.T) {
	prov := []artifact.Provenance{
		{
			ItemIndex: 1,
			ItemType:  "message",
			Spans: []artifact.ProvenanceSpan{
				{
					ContentStart: 0,
					ContentEnd:   10,
					Trace: []artifact.TraceSpan{
						{RequestID: "req-1", EventIndex: 1, EventType: "delta", ByteStart: 0, ByteEnd: 5},
					},
				},
			},
		},
	}

	art1 := newArtifact("a1", "s1", "t1", "r1", artifact.ArtifactDecision, "Decision: use sqlite.")
	art1.Provenance = prov
	art2 := newArtifact("a2", "s1", "t1", "r1", artifact.ArtifactTranscript, "We will index artifacts.")
	art3 := newArtifact("a3", "s1", "t2", "r2", artifact.ArtifactOutcome, "Outcome: done.")
	art3.Provenance = prov
	art4 := newArtifact("a4", "s2", "t3", "r3", artifact.ArtifactOutcome, "Unrelated")

	idx := buildIndex(t, []artifact.Artifact{art1, art2, art3, art4}, map[string]time.Time{
		"a1": time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		"a2": time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		"a3": time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC),
		"a4": time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
	})

	cards, err := idx.Related("a1", RelatedOptions{Limit: 5, SnippetMaxLen: 60})
	if err != nil {
		t.Fatalf("related: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 related cards, got %d", len(cards))
	}
	if cards[0].ArtifactID != "a2" {
		t.Fatalf("expected run/task neighbor first, got %s", cards[0].ArtifactID)
	}
	ids := map[string]bool{}
	for _, card := range cards {
		ids[card.ArtifactID] = true
	}
	if !ids["a2"] || !ids["a3"] {
		t.Fatalf("expected related ids a2 and a3, got %+v", ids)
	}
	if ids["a4"] {
		t.Fatalf("unexpected unrelated artifact in related results")
	}
}
