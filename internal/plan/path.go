package plan

import (
	"os"
	"path/filepath"
)

// PlanPath returns the plan file path for the current working directory.
func PlanPath() string {
	wd, err := os.Getwd()
	if err != nil {
		// If this fails, other file ops will fail too; keep path deterministic.
		return DefaultPlanFilename
	}
	return filepath.Join(wd, DefaultPlanFilename)
}
