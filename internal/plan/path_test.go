package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanPathUsesWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})

	tmpResolved := tmp
	if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
		tmpResolved = resolved
	}

	want := filepath.Join(tmpResolved, DefaultPlanFilename)
	got := PlanPath()
	gotDir := filepath.Dir(got)
	if resolved, err := filepath.EvalSymlinks(gotDir); err == nil {
		got = filepath.Join(resolved, DefaultPlanFilename)
	}
	if got != want {
		t.Fatalf("PlanPath() = %q, want %q", got, want)
	}
}
