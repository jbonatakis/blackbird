package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

// Get returns a full artifact by ID.
func (idx *Index) Get(artifactID string) (artifact.Artifact, bool, error) {
	if idx == nil || idx.db == nil {
		return artifact.Artifact{}, false, fmt.Errorf("index not initialized")
	}
	if artifactID == "" {
		return artifact.Artifact{}, false, fmt.Errorf("artifact id required")
	}

	row := idx.db.QueryRowContext(context.Background(), "SELECT artifact_json FROM artifacts WHERE id = ?", artifactID)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return artifact.Artifact{}, false, nil
		}
		return artifact.Artifact{}, false, fmt.Errorf("load artifact: %w", err)
	}
	var art artifact.Artifact
	if err := json.Unmarshal([]byte(payload), &art); err != nil {
		return artifact.Artifact{}, false, fmt.Errorf("decode artifact: %w", err)
	}
	return art, true, nil
}
