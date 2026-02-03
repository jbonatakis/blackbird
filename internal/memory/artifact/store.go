package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/canonical"
)

// LoadStore reads the artifact store from path. Returns false if not found.
func LoadStore(path string) (Store, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Store{}, false, nil
		}
		return Store{}, false, fmt.Errorf("read artifact store: %w", err)
	}
	return decodeStore(payload)
}

// LoadStoreForProject loads the artifact store for the given project root.
func LoadStoreForProject(projectRoot string) (Store, bool, error) {
	path := memory.ArtifactsDBPath(projectRoot)
	return LoadStore(path)
}

// SaveStore writes the artifact store to path.
func SaveStore(path string, store Store) error {
	if store.SchemaVersion == 0 {
		store.SchemaVersion = SchemaVersion
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode artifact store: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create artifact store dir: %w", err)
	}
	if err := memory.AtomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write artifact store: %w", err)
	}
	return nil
}

// SaveStoreForProject saves the artifact store for the given project root.
func SaveStoreForProject(projectRoot string, store Store) error {
	path := memory.ArtifactsDBPath(projectRoot)
	return SaveStore(path, store)
}

// UpdateStore merges artifacts extracted from logs into the store and saves it.
func UpdateStore(projectRoot string, logs []canonical.Log) (Store, error) {
	existing, _, err := LoadStoreForProject(projectRoot)
	if err != nil {
		return Store{}, err
	}
	updated := Store{
		SchemaVersion: SchemaVersion,
		Artifacts:     BuildArtifacts(existing.Artifacts, logs),
	}
	if err := SaveStoreForProject(projectRoot, updated); err != nil {
		return Store{}, err
	}
	return updated, nil
}

func decodeStore(payload []byte) (Store, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	var store Store
	if err := dec.Decode(&store); err != nil {
		return Store{}, true, fmt.Errorf("decode artifact store: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Store{}, true, errors.New("decode artifact store: trailing JSON values")
		}
		return Store{}, true, fmt.Errorf("decode artifact store: trailing data: %w", err)
	}
	if store.SchemaVersion == 0 {
		store.SchemaVersion = SchemaVersion
	}
	return store, true, nil
}
