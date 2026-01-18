package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad_RoundTripEmptyGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultPlanFilename)

	g0 := NewEmptyWorkGraph()
	if err := SaveAtomic(path, g0); err != nil {
		t.Fatalf("SaveAtomic: %v", err)
	}

	g1, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if g1.SchemaVersion != SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", g1.SchemaVersion, SchemaVersion)
	}
	if g1.Items == nil {
		t.Fatalf("items is nil")
	}
	if errs := Validate(g1); len(errs) != 0 {
		t.Fatalf("Validate after round-trip: %v", errs)
	}
}

func TestLoad_MissingReturnsSentinel(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrPlanNotFound {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
	_ = os.ErrNotExist // silence vet about unused os import if it changes later
}
