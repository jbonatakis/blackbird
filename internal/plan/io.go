package plan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrPlanNotFound = errors.New("plan file not found")

func Load(path string) (WorkGraph, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return WorkGraph{}, ErrPlanNotFound
		}
		return WorkGraph{}, fmt.Errorf("read plan file %s: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	var g WorkGraph
	if err := dec.Decode(&g); err != nil {
		return WorkGraph{}, fmt.Errorf("parse plan file %s: %w", path, err)
	}
	// Ensure there's nothing but whitespace after the object.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return WorkGraph{}, fmt.Errorf("parse plan file %s: trailing JSON values", path)
		}
		return WorkGraph{}, fmt.Errorf("parse plan file %s: trailing data: %w", path, err)
	}
	return g, nil
}

func SaveAtomic(path string, g WorkGraph) error {
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	b = append(b, '\n')
	return atomicWriteFile(path, b, 0o644)
}
