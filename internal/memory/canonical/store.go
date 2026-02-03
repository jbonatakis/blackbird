package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbonatakis/blackbird/internal/memory"
)

func SaveLog(projectRoot string, log Log) error {
	if strings.TrimSpace(log.RunID) == "" {
		return errors.New("run id is required to save canonical log")
	}
	if log.SchemaVersion == 0 {
		log.SchemaVersion = SchemaVersion
	}
	if err := ValidateLog(log); err != nil {
		return err
	}
	path := memory.CanonicalLogPath(projectRoot, log.RunID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create canonical log dir: %w", err)
	}
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("encode canonical log: %w", err)
	}
	data = append(data, '\n')
	if err := memory.AtomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write canonical log: %w", err)
	}
	return nil
}

func SaveLogs(projectRoot string, logs []Log) error {
	for _, log := range logs {
		if strings.TrimSpace(log.RunID) == "" {
			continue
		}
		if err := SaveLog(projectRoot, log); err != nil {
			return err
		}
	}
	return nil
}

func LoadLog(projectRoot string, runID string) (Log, bool, error) {
	if strings.TrimSpace(runID) == "" {
		return Log{}, false, errors.New("run id is required")
	}
	path := memory.CanonicalLogPath(projectRoot, runID)
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Log{}, false, nil
		}
		return Log{}, false, fmt.Errorf("read canonical log: %w", err)
	}
	return decodeLog(payload)
}

func decodeLog(payload []byte) (Log, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()

	var log Log
	if err := dec.Decode(&log); err != nil {
		return Log{}, true, fmt.Errorf("decode canonical log: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Log{}, true, errors.New("decode canonical log: trailing JSON values")
		}
		return Log{}, true, fmt.Errorf("decode canonical log: trailing data: %w", err)
	}

	if log.SchemaVersion == 0 {
		log.SchemaVersion = SchemaVersion
	}
	if err := ValidateLog(log); err != nil {
		return Log{}, true, err
	}
	return log, true, nil
}
