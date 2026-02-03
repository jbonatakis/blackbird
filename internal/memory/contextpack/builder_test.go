package contextpack

import (
	"reflect"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

func TestBuildContextPackOrdering(t *testing.T) {
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	artifacts := []artifact.Artifact{
		newArtifact("d1", "s1", "t1", "r1", artifact.ArtifactDecision, artifact.Content{Text: "Use sqlite"}, []string{"d2"}),
		newArtifact("d2", "s1", "t1", "r2", artifact.ArtifactDecision, artifact.Content{Text: "Use sqlite", Rationale: "lower deps"}, nil),
		newArtifact("d3", "s1", "t2", "r3", artifact.ArtifactDecision, artifact.Content{Text: "Use postgres"}, nil),
		newArtifact("c1", "s1", "t1", "r4", artifact.ArtifactConstraint, artifact.Content{Text: "Avoid destructive commands"}, nil),
		newArtifact("c2", "s1", "t2", "r5", artifact.ArtifactConstraint, artifact.Content{Text: "Prefer go"}, nil),
		newArtifact("o1", "s1", "t1", "r6", artifact.ArtifactOutcome, artifact.Content{Status: "success", Summary: []string{"Initial setup"}}, nil),
		newArtifact("o2", "s1", "t1", "r7", artifact.ArtifactOutcome, artifact.Content{Status: "success", Summary: []string{"Fixed bug"}}, nil),
		newArtifact("o3", "s1", "t2", "r8", artifact.ArtifactOutcome, artifact.Content{Status: "success", Summary: []string{"Implemented builder"}}, nil),
		newArtifact("ot1", "s1", "t1", "r9", artifact.ArtifactOpenThread, artifact.Content{Text: "Need UI decision"}, nil),
		newArtifact("ot2", "s1", "t1", "r10", artifact.ArtifactOpenThread, artifact.Content{Text: "Need UI decision"}, nil),
		newArtifact("ot3", "s1", "t2", "r11", artifact.ArtifactOpenThread, artifact.Content{Text: "Write tests"}, nil),
	}

	runTimes := map[string]time.Time{
		"t1:r1":  now.Add(-180 * time.Minute),
		"t1:r2":  now.Add(-120 * time.Minute),
		"t2:r3":  now.Add(-70 * time.Minute),
		"t1:r4":  now.Add(-240 * time.Minute),
		"t2:r5":  now.Add(-30 * time.Minute),
		"t1:r6":  now.Add(-180 * time.Minute),
		"t1:r7":  now.Add(-60 * time.Minute),
		"t2:r8":  now.Add(-20 * time.Minute),
		"t1:r9":  now.Add(-300 * time.Minute),
		"t1:r10": now.Add(-10 * time.Minute),
		"t2:r11": now.Add(-15 * time.Minute),
	}

	pack := Build(BuildOptions{
		SessionID:     "s1",
		SessionGoal:   "Build memory context pack",
		Artifacts:     artifacts,
		Budget:        Budget{TotalTokens: 1000, DecisionsTokens: 1000, ConstraintsTokens: 1000, ImplementedTokens: 1000, OpenThreadsTokens: 1000, ArtifactPointersTokens: 1000},
		RunTimeLookup: lookupFromMap(runTimes),
		Now:           now,
	})

	wantDecisions := []string{
		"Use postgres [id: d3]",
		"Use sqlite (rationale: lower deps) [id: d2]",
	}
	if !reflect.DeepEqual(pack.Decisions.Items, wantDecisions) {
		t.Fatalf("decisions = %#v, want %#v", pack.Decisions.Items, wantDecisions)
	}

	wantConstraints := []string{
		"Prefer go [id: c2]",
		"Avoid destructive commands [id: c1]",
	}
	if !reflect.DeepEqual(pack.Constraints.Items, wantConstraints) {
		t.Fatalf("constraints = %#v, want %#v", pack.Constraints.Items, wantConstraints)
	}

	wantOutcomes := []string{
		"t2 (success): Implemented builder [id: o3]",
		"t1 (success): Fixed bug [id: o2]",
	}
	if !reflect.DeepEqual(pack.Implemented.Items, wantOutcomes) {
		t.Fatalf("outcomes = %#v, want %#v", pack.Implemented.Items, wantOutcomes)
	}

	wantThreads := []string{
		"Need UI decision [id: ot2]",
		"Write tests [id: ot3]",
	}
	if !reflect.DeepEqual(pack.OpenThreads.Items, wantThreads) {
		t.Fatalf("open threads = %#v, want %#v", pack.OpenThreads.Items, wantThreads)
	}

	wantArtifacts := []string{
		"ot2 (open_thread)",
		"ot3 (open_thread)",
		"o3 (outcome)",
		"c2 (constraint)",
		"o2 (outcome)",
		"d3 (decision)",
		"d2 (decision)",
		"c1 (constraint)",
	}
	if !reflect.DeepEqual(pack.ArtifactIDs.Items, wantArtifacts) {
		t.Fatalf("artifact ids = %#v, want %#v", pack.ArtifactIDs.Items, wantArtifacts)
	}
}

func TestBuildContextPackBudgeting(t *testing.T) {
	now := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	artifacts := []artifact.Artifact{
		newArtifact("d1", "s1", "t1", "r1", artifact.ArtifactDecision, artifact.Content{Text: "alpha beta"}, nil),
		newArtifact("d2", "s1", "t1", "r2", artifact.ArtifactDecision, artifact.Content{Text: "gamma delta"}, nil),
	}
	runTimes := map[string]time.Time{
		"t1:r1": now.Add(-20 * time.Minute),
		"t1:r2": now.Add(-10 * time.Minute),
	}

	pack := Build(BuildOptions{
		SessionID:     "s1",
		Artifacts:     artifacts,
		Budget:        Budget{TotalTokens: 200, DecisionsTokens: 4},
		RunTimeLookup: lookupFromMap(runTimes),
		Now:           now,
	})

	wantDecisions := []string{"gamma delta [id: d2]"}
	if !reflect.DeepEqual(pack.Decisions.Items, wantDecisions) {
		t.Fatalf("decisions = %#v, want %#v", pack.Decisions.Items, wantDecisions)
	}
	if pack.Decisions.Tokens > 4 {
		t.Fatalf("decisions tokens = %d, want <= 4", pack.Decisions.Tokens)
	}
}

func lookupFromMap(values map[string]time.Time) RunTimeLookup {
	return func(taskID, runID string) (time.Time, bool) {
		if taskID == "" || runID == "" {
			return time.Time{}, false
		}
		key := taskID + ":" + runID
		ts, ok := values[key]
		return ts, ok
	}
}

func newArtifact(id, sessionID, taskID, runID string, typ artifact.ArtifactType, content artifact.Content, supersededBy []string) artifact.Artifact {
	return artifact.Artifact{
		SchemaVersion:  artifact.SchemaVersion,
		ArtifactID:     id,
		SessionID:      sessionID,
		TaskID:         taskID,
		RunID:          runID,
		Type:           typ,
		Content:        content,
		SupersededBy:   supersededBy,
		BuilderVersion: artifact.BuilderVersion,
	}
}
