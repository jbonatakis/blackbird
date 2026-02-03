package proxy

import (
	"io"
	"sync"
)

type bodyRecorder struct {
	rc      io.ReadCloser
	onChunk func(seq int, data []byte)

	mu    sync.Mutex
	seq   int
	total int64
}

func newBodyRecorder(rc io.ReadCloser, onChunk func(seq int, data []byte)) *bodyRecorder {
	if rc == nil {
		return nil
	}
	return &bodyRecorder{rc: rc, onChunk: onChunk}
}

func (b *bodyRecorder) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	if n > 0 {
		chunk := make([]byte, n)
		copy(chunk, p[:n])
		b.mu.Lock()
		b.seq++
		seq := b.seq
		b.total += int64(n)
		b.mu.Unlock()
		if b.onChunk != nil {
			b.onChunk(seq, chunk)
		}
	}
	return n, err
}

func (b *bodyRecorder) Close() error {
	if b == nil || b.rc == nil {
		return nil
	}
	return b.rc.Close()
}

func (b *bodyRecorder) Bytes() int64 {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total
}
