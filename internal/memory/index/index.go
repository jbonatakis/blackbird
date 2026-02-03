package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"

	_ "modernc.org/sqlite"
)

// Index manages the SQLite-backed lexical index.
type Index struct {
	db *sql.DB
}

// Open opens (or creates) an index at path.
func Open(path string) (*Index, error) {
	if path == "" {
		return nil, fmt.Errorf("index path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	idx := &Index{db: db}
	if err := idx.ensureSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return idx, nil
}

// Close closes the index database.
func (idx *Index) Close() error {
	if idx == nil || idx.db == nil {
		return nil
	}
	return idx.db.Close()
}

// Rebuild clears and rebuilds the index from artifacts.
func (idx *Index) Rebuild(artifacts []artifact.Artifact, opts RebuildOptions) error {
	if idx == nil || idx.db == nil {
		return fmt.Errorf("index not initialized")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	ctx := context.Background()
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rebuild: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, "DELETE FROM artifact_links"); err != nil {
		return fmt.Errorf("clear links: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM artifacts"); err != nil {
		return fmt.Errorf("clear artifacts: %w", err)
	}

	insertArtifact, err := tx.PrepareContext(ctx, `
		INSERT INTO artifacts
		(id, session_id, task_id, run_id, type, created_at, text, provenance_json, artifact_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare artifact insert: %w", err)
	}
	defer insertArtifact.Close()

	insertLink, err := tx.PrepareContext(ctx, `
		INSERT INTO artifact_links (artifact_id, link_type, link_value)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare link insert: %w", err)
	}
	defer insertLink.Close()

	for _, art := range artifacts {
		createdAt := resolveArtifactTime(art, opts)
		text := indexText(art)
		provenanceJSON, err := json.Marshal(art.Provenance)
		if err != nil {
			return fmt.Errorf("encode provenance: %w", err)
		}
		artifactJSON, err := json.Marshal(art)
		if err != nil {
			return fmt.Errorf("encode artifact: %w", err)
		}
		if _, err := insertArtifact.ExecContext(
			ctx,
			art.ArtifactID,
			art.SessionID,
			art.TaskID,
			art.RunID,
			string(art.Type),
			createdAt.Unix(),
			text,
			string(provenanceJSON),
			string(artifactJSON),
		); err != nil {
			return fmt.Errorf("insert artifact: %w", err)
		}
		for link := range linkKeys(art) {
			if _, err := insertLink.ExecContext(ctx, art.ArtifactID, link.linkType, link.value); err != nil {
				return fmt.Errorf("insert link: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rebuild: %w", err)
	}
	rollback = false

	if _, err := idx.db.ExecContext(ctx, "INSERT INTO artifacts_fts(artifacts_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("rebuild fts: %w", err)
	}
	if _, err := idx.db.ExecContext(ctx, "INSERT OR REPLACE INTO index_meta(key, value) VALUES (?, ?)", metaKeySchemaVersion, schemaVersion); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}
	return nil
}

// RebuildForProject loads the artifact store for the project and rebuilds the index.
func RebuildForProject(projectRoot string, opts RebuildOptions) error {
	store, _, err := artifact.LoadStoreForProject(projectRoot)
	if err != nil {
		return err
	}
	if opts.RunTimeLookup == nil {
		opts.RunTimeLookup = RunTimeLookupFromExecution(projectRoot)
	}
	idx, err := Open(memory.IndexDBPath(projectRoot))
	if err != nil {
		return err
	}
	defer idx.Close()
	return idx.Rebuild(store.Artifacts, opts)
}

// RunTimeLookupFromExecution uses execution run records for timestamps.
func RunTimeLookupFromExecution(baseDir string) RunTimeLookup {
	cache := make(map[string]time.Time)
	return func(taskID, runID string) (time.Time, bool) {
		if taskID == "" || runID == "" {
			return time.Time{}, false
		}
		key := taskID + ":" + runID
		if ts, ok := cache[key]; ok {
			return ts, true
		}
		record, err := execution.LoadRun(baseDir, taskID, runID)
		if err != nil {
			return time.Time{}, false
		}
		cache[key] = record.StartedAt
		return record.StartedAt, true
	}
}

func (idx *Index) ensureSchema(ctx context.Context) error {
	if _, err := idx.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func resolveArtifactTime(art artifact.Artifact, opts RebuildOptions) time.Time {
	if opts.TimestampFor != nil {
		timeValue := opts.TimestampFor(art)
		if !timeValue.IsZero() {
			return timeValue
		}
	}
	if opts.RunTimeLookup != nil {
		if ts, ok := opts.RunTimeLookup(art.TaskID, art.RunID); ok {
			return ts
		}
	}
	if !opts.Now.IsZero() {
		return opts.Now
	}
	return time.Now()
}
