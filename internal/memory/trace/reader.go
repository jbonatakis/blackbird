package trace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Reader struct {
	paths []string
	idx   int

	file   *os.File
	reader *bufio.Reader
}

func NewReader(path string) (*Reader, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("wal path is required")
	}
	dir := filepath.Dir(path)
	baseName := strings.TrimSuffix(filepath.Base(path), rotatedSuffix)
	if baseName == "" {
		return nil, fmt.Errorf("wal base name is required")
	}
	paths, err := listWALFiles(dir, baseName)
	if err != nil {
		return nil, err
	}
	return &Reader{paths: paths}, nil
}

func (r *Reader) Next() (Event, bool, error) {
	for {
		if r.reader == nil {
			if err := r.openNext(); err != nil {
				if err == io.EOF {
					return Event{}, false, nil
				}
				return Event{}, false, err
			}
		}

		line, err := r.reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return Event{}, false, fmt.Errorf("read wal: %w", err)
		}
		if err == io.EOF {
			r.closeCurrent()
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if err == io.EOF {
				continue
			}
			continue
		}

		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return Event{}, false, fmt.Errorf("parse event: %w", err)
		}
		return ev, true, nil
	}
}

func Replay(path string) ([]Event, error) {
	reader, err := NewReader(path)
	if err != nil {
		return nil, err
	}
	var events []Event
	for {
		ev, ok, err := reader.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *Reader) openNext() error {
	if r.idx >= len(r.paths) {
		return io.EOF
	}
	path := r.paths[r.idx]
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open wal %s: %w", path, err)
	}
	r.idx++
	r.file = file
	r.reader = bufio.NewReader(file)
	return nil
}

func (r *Reader) closeCurrent() {
	if r.file != nil {
		_ = r.file.Close()
	}
	r.file = nil
	r.reader = nil
}

func listWALFiles(dir string, baseName string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wal dir: %w", err)
	}

	activeName := baseName + rotatedSuffix
	var activePath string
	type rotated struct {
		path string
		when time.Time
	}
	rotatedFiles := []rotated{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == activeName {
			activePath = filepath.Join(dir, name)
			continue
		}
		if !strings.HasPrefix(name, baseName+"-") || !strings.HasSuffix(name, rotatedSuffix) {
			continue
		}
		when := timestampFromName(baseName, name)
		if when.IsZero() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			when = info.ModTime()
		}
		rotatedFiles = append(rotatedFiles, rotated{
			path: filepath.Join(dir, name),
			when: when,
		})
	}

	sort.Slice(rotatedFiles, func(i, j int) bool {
		if rotatedFiles[i].when.Equal(rotatedFiles[j].when) {
			return rotatedFiles[i].path < rotatedFiles[j].path
		}
		return rotatedFiles[i].when.Before(rotatedFiles[j].when)
	})

	paths := make([]string, 0, len(rotatedFiles)+1)
	for _, entry := range rotatedFiles {
		paths = append(paths, entry.path)
	}
	if activePath != "" {
		paths = append(paths, activePath)
	}

	return paths, nil
}
