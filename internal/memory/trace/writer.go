package trace

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const DefaultMaxAge = 24 * time.Hour

const (
	rotatedSuffix = ".wal"
)

type Options struct {
	MaxSizeBytes int64
	MaxAge       time.Duration
	Retention    time.Duration
	PrivacyMode  bool
	FsyncOnWrite bool
	// FsyncOnWriteSet lets callers explicitly disable fsync-on-append.
	FsyncOnWriteSet bool
	Redactor        *Redactor
	Now             func() time.Time
}

type WALWriter struct {
	mu           sync.Mutex
	path         string
	dir          string
	baseName     string
	file         *os.File
	buf          *bufio.Writer
	bytesWritten int64
	openedAt     time.Time
	opts         Options
}

func NewWALWriter(path string, opts Options) (*WALWriter, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("wal path is required")
	}
	opts = applyDefaults(opts)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(path), rotatedSuffix)
	if baseName == "" {
		return nil, errors.New("wal base name is required")
	}

	file, buf, size, openedAt, err := openWAL(path, opts.Now())
	if err != nil {
		return nil, err
	}

	writer := &WALWriter{
		path:         path,
		dir:          dir,
		baseName:     baseName,
		file:         file,
		buf:          buf,
		bytesWritten: size,
		openedAt:     openedAt,
		opts:         opts,
	}

	return writer, nil
}

func (w *WALWriter) Append(event Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil || w.buf == nil {
		return errors.New("wal writer is closed")
	}

	if isBodyEvent(event.Type) && w.opts.PrivacyMode {
		return nil
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = w.opts.Now()
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = SchemaVersion
	}
	if event.Headers != nil {
		event.Headers = w.opts.Redactor.RedactHeaders(event.Headers)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	payload = append(payload, '\n')

	if w.shouldRotate(int64(len(payload))) {
		if err := w.rotate(); err != nil {
			return err
		}
	}

	if _, err := w.buf.Write(payload); err != nil {
		return fmt.Errorf("write wal: %w", err)
	}
	w.bytesWritten += int64(len(payload))

	if w.opts.FsyncOnWrite {
		if err := w.flushAndSync(); err != nil {
			return err
		}
	}

	return nil
}

func (w *WALWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	if err := w.flushAndSync(); err != nil {
		return err
	}
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close wal: %w", err)
	}
	w.file = nil
	w.buf = nil
	return nil
}

func (w *WALWriter) shouldRotate(nextWrite int64) bool {
	if w.file == nil {
		return false
	}
	if w.bytesWritten == 0 {
		return false
	}

	if w.opts.MaxSizeBytes > 0 && w.bytesWritten+nextWrite > w.opts.MaxSizeBytes {
		return true
	}
	if w.opts.MaxAge > 0 {
		age := w.opts.Now().Sub(w.openedAt)
		if age >= w.opts.MaxAge {
			return true
		}
	}
	return false
}

func (w *WALWriter) rotate() error {
	if w.file == nil {
		return nil
	}

	if err := w.flushAndSync(); err != nil {
		return err
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close wal before rotate: %w", err)
	}

	rotatedName := rotationName(w.baseName, w.opts.Now())
	rotatedPath := filepath.Join(w.dir, rotatedName)
	if err := os.Rename(w.path, rotatedPath); err != nil {
		return fmt.Errorf("rotate wal: %w", err)
	}
	if err := fsyncDir(w.dir); err != nil {
		return fmt.Errorf("fsync wal dir: %w", err)
	}

	file, buf, size, openedAt, err := openWAL(w.path, w.opts.Now())
	if err != nil {
		return err
	}
	w.file = file
	w.buf = buf
	w.bytesWritten = size
	w.openedAt = openedAt

	if err := pruneRetention(w.dir, w.baseName, w.opts.Retention, w.opts.Now()); err != nil {
		return err
	}

	return nil
}

func (w *WALWriter) flushAndSync() error {
	if w.buf == nil || w.file == nil {
		return nil
	}
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("flush wal: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("fsync wal: %w", err)
	}
	return nil
}

func openWAL(path string, now time.Time) (*os.File, *bufio.Writer, int64, time.Time, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, 0, time.Time{}, fmt.Errorf("open wal: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, 0, time.Time{}, fmt.Errorf("stat wal: %w", err)
	}
	openedAt := info.ModTime()
	if info.Size() == 0 {
		openedAt = now
	}
	buf := bufio.NewWriter(file)
	return file, buf, info.Size(), openedAt, nil
}

func applyDefaults(opts Options) Options {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Redactor == nil {
		opts.Redactor = DefaultRedactor()
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = DefaultMaxAge
	}
	if opts.MaxAge < 0 {
		opts.MaxAge = 0
	}
	if !opts.FsyncOnWriteSet {
		opts.FsyncOnWrite = true
	}
	return opts
}

func isBodyEvent(eventType string) bool {
	if strings.HasPrefix(eventType, "request.body") {
		return true
	}
	if strings.HasPrefix(eventType, "response.body") {
		return true
	}
	return false
}

func rotationName(baseName string, now time.Time) string {
	return fmt.Sprintf("%s-%d%s", baseName, now.UTC().UnixNano(), rotatedSuffix)
}

func pruneRetention(dir string, baseName string, retention time.Duration, now time.Time) error {
	if retention <= 0 {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read wal dir: %w", err)
	}
	cutoff := now.Add(-retention)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == baseName+rotatedSuffix {
			continue
		}
		if !strings.HasPrefix(name, baseName+"-") || !strings.HasSuffix(name, rotatedSuffix) {
			continue
		}
		path := filepath.Join(dir, name)
		when := timestampFromName(baseName, name)
		if when.IsZero() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			when = info.ModTime()
		}
		if when.Before(cutoff) {
			_ = os.Remove(path)
		}
	}
	return nil
}

func timestampFromName(baseName string, name string) time.Time {
	trimmed := strings.TrimPrefix(name, baseName+"-")
	trimmed = strings.TrimSuffix(trimmed, rotatedSuffix)
	if trimmed == "" {
		return time.Time{}
	}
	nanos, err := parseInt64(trimmed)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, nanos).UTC()
}

func parseInt64(value string) (int64, error) {
	var out int64
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid number")
		}
		out = out*10 + int64(ch-'0')
	}
	return out, nil
}

func fsyncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
