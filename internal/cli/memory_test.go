package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
	"github.com/jbonatakis/blackbird/internal/memory/index"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunMemSearchFiltersLimitAndFormats(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	artifacts := []artifact.Artifact{
		newArtifact("a1", "s1", "t1", "r1", artifact.ArtifactDecision, "alpha old"),
		newArtifact("a2", "s1", "t1", "r2", artifact.ArtifactDecision, "alpha new"),
		newArtifact("a3", "s2", "t1", "r3", artifact.ArtifactDecision, "alpha other"),
	}

	idx, err := index.Open(memory.IndexDBPath(tempDir))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer func() { _ = idx.Close() }()

	times := map[string]time.Time{
		"a1": now.Add(-2 * time.Hour),
		"a2": now.Add(-1 * time.Hour),
		"a3": now.Add(-3 * time.Hour),
	}
	if err := idx.Rebuild(artifacts, index.RebuildOptions{
		TimestampFor: func(art artifact.Artifact) time.Time {
			if ts, ok := times[art.ArtifactID]; ok {
				return ts
			}
			return now
		},
	}); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	output, err := captureStdout(func() error {
		return runMemSearch([]string{"--session", "s1", "--task", "t1", "--limit", "1", "alpha"})
	})
	if err != nil {
		t.Fatalf("runMemSearch: %v", err)
	}
	if !strings.Contains(output, "Artifact ID") || !strings.Contains(output, "Snippet") {
		t.Fatalf("expected table header, got: %q", output)
	}
	if !strings.Contains(output, "a2") {
		t.Fatalf("expected newest artifact a2, got: %q", output)
	}
	if strings.Contains(output, "a1") || strings.Contains(output, "a3") {
		t.Fatalf("expected filters/limit to exclude other artifacts, got: %q", output)
	}
}

func TestRunMemSearchRequiresQuery(t *testing.T) {
	if err := runMemSearch([]string{"--session", "s1"}); err == nil {
		t.Fatalf("expected error for missing query")
	} else if !strings.Contains(err.Error(), "query") {
		t.Fatalf("expected query error, got: %v", err)
	}
}

func TestRunMemGetOutputsArtifactJSON(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	artifacts := []artifact.Artifact{
		newArtifact("a1", "s1", "t1", "r1", artifact.ArtifactDecision, "use sqlite"),
	}

	idx, err := index.Open(memory.IndexDBPath(tempDir))
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer func() { _ = idx.Close() }()
	if err := idx.Rebuild(artifacts, index.RebuildOptions{}); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	output, err := captureStdout(func() error {
		return runMemGet([]string{"a1"})
	})
	if err != nil {
		t.Fatalf("runMemGet: %v", err)
	}
	if !strings.Contains(output, "\"artifact_id\": \"a1\"") {
		t.Fatalf("expected artifact id in output, got: %q", output)
	}
	if !strings.Contains(output, "\"type\": \"decision\"") {
		t.Fatalf("expected artifact type in output, got: %q", output)
	}
}

func TestRunMemContextRendersPack(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": newWorkItem("task-1", now),
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	session, err := memory.CreateSession(memory.SessionPath(tempDir), "Build memory pack")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	store := artifact.Store{
		SchemaVersion: artifact.SchemaVersion,
		Artifacts: []artifact.Artifact{
			newArtifact("d1", session.SessionID, "task-1", "run-1", artifact.ArtifactDecision, "Use sqlite"),
		},
	}
	if err := artifact.SaveStoreForProject(tempDir, store); err != nil {
		t.Fatalf("save store: %v", err)
	}

	output, err := captureStdout(func() error {
		return runMemContext([]string{"--task", "task-1"})
	})
	if err != nil {
		t.Fatalf("runMemContext: %v", err)
	}
	if !strings.Contains(output, "Session context pack") {
		t.Fatalf("expected context pack header, got: %q", output)
	}
	if !strings.Contains(output, session.SessionID) {
		t.Fatalf("expected session id in output, got: %q", output)
	}
	if !strings.Contains(output, "Decisions:") {
		t.Fatalf("expected decisions section, got: %q", output)
	}
	if !strings.Contains(output, "Use sqlite") {
		t.Fatalf("expected decision text, got: %q", output)
	}
}

func TestRunMemContextRequiresTask(t *testing.T) {
	if err := runMemContext([]string{}); err == nil {
		t.Fatalf("expected error for missing task")
	} else if !strings.Contains(err.Error(), "task") {
		t.Fatalf("expected task error, got: %v", err)
	}
}

func newArtifact(id, sessionID, taskID, runID string, typ artifact.ArtifactType, text string) artifact.Artifact {
	return artifact.Artifact{
		SchemaVersion:  artifact.SchemaVersion,
		ArtifactID:     id,
		SessionID:      sessionID,
		TaskID:         taskID,
		RunID:          runID,
		Type:           typ,
		Content:        artifact.Content{Text: text},
		BuilderVersion: artifact.BuilderVersion,
	}
}
