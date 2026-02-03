package memory

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SessionSchemaVersion = 1

type Session struct {
	SchemaVersion int    `json:"schemaVersion"`
	SessionID     string `json:"session_id"`
	Goal          string `json:"goal,omitempty"`
}

// LoadSession reads session metadata from disk.
func LoadSession(path string) (Session, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, false, nil
		}
		return Session{}, false, fmt.Errorf("read session %s: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	var session Session
	if err := dec.Decode(&session); err != nil {
		return Session{}, true, fmt.Errorf("parse session %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Session{}, true, fmt.Errorf("parse session %s: trailing JSON values", path)
		}
		return Session{}, true, fmt.Errorf("parse session %s: trailing data: %w", path, err)
	}

	if session.SchemaVersion != SessionSchemaVersion {
		return Session{}, true, fmt.Errorf("unsupported session schema version %d", session.SchemaVersion)
	}
	session.SessionID = strings.TrimSpace(session.SessionID)
	if session.SessionID == "" {
		return Session{}, true, errors.New("session_id is required")
	}
	session.Goal = strings.TrimSpace(session.Goal)

	return session, true, nil
}

// CreateSession generates and persists a new session metadata file.
func CreateSession(path string, goal string) (Session, error) {
	id, err := newSessionID()
	if err != nil {
		return Session{}, fmt.Errorf("generate session id: %w", err)
	}
	session := Session{
		SchemaVersion: SessionSchemaVersion,
		SessionID:     id,
		Goal:          strings.TrimSpace(goal),
	}

	b, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return Session{}, fmt.Errorf("encode session: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Session{}, fmt.Errorf("create session dir: %w", err)
	}

	if err := atomicWriteFile(path, b, 0o644); err != nil {
		return Session{}, fmt.Errorf("write session %s: %w", path, err)
	}

	return session, nil
}

// LoadOrCreateSession loads session metadata or creates a new session if missing.
func LoadOrCreateSession(path string, goal string) (Session, bool, error) {
	session, present, err := LoadSession(path)
	if err != nil {
		return Session{}, false, err
	}
	if present {
		return session, false, nil
	}
	createdSession, err := CreateSession(path, goal)
	if err != nil {
		return Session{}, false, err
	}
	return createdSession, true, nil
}

func newSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
