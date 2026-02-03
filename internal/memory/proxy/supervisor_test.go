package proxy

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/jbonatakis/blackbird/internal/config"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
)

func TestSupervisorStartNoopForUnsupportedProvider(t *testing.T) {
	supervisor := NewSupervisor()
	handle, err := supervisor.Start(SupervisorOptions{ProviderID: "unsupported"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running, refCount := supervisorState(supervisor)
	if running || refCount != 0 {
		t.Fatalf("expected no-op start, running=%v refCount=%d", running, refCount)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestSupervisorStartNoopWhenMemoryDisabled(t *testing.T) {
	supervisor := NewSupervisor()
	cfg := config.DefaultResolvedConfig()
	cfg.Memory.Mode = config.MemoryModeOff

	handle, err := supervisor.Start(SupervisorOptions{
		ProviderID:  memprovider.ProviderCodex,
		ProjectRoot: t.TempDir(),
		Config:      &cfg,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running, refCount := supervisorState(supervisor)
	if running || refCount != 0 {
		t.Fatalf("expected memory-off no-op, running=%v refCount=%d", running, refCount)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestSupervisorStartAndShutdown(t *testing.T) {
	supervisor := NewSupervisor()
	cfg := config.DefaultResolvedConfig()
	cfg.Memory.Proxy.ListenAddr = "127.0.0.1:0"
	listener := newStubListener("127.0.0.1:0")

	handle1, err := supervisor.Start(SupervisorOptions{
		ProviderID:  memprovider.ProviderCodex,
		ProjectRoot: t.TempDir(),
		Config:      &cfg,
		ListenFunc: func(network, addr string) (net.Listener, error) {
			return listener, nil
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	running, refCount := supervisorState(supervisor)
	if !running || refCount != 1 {
		t.Fatalf("expected running refCount=1, got running=%v refCount=%d", running, refCount)
	}

	handle2, err := supervisor.Start(SupervisorOptions{
		ProviderID:  memprovider.ProviderCodex,
		ProjectRoot: t.TempDir(),
		Config:      &cfg,
		ListenFunc: func(network, addr string) (net.Listener, error) {
			return listener, nil
		},
	})
	if err != nil {
		t.Fatalf("Start() second error = %v", err)
	}
	running, refCount = supervisorState(supervisor)
	if !running || refCount != 2 {
		t.Fatalf("expected running refCount=2, got running=%v refCount=%d", running, refCount)
	}

	if err := handle1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	running, refCount = supervisorState(supervisor)
	if !running || refCount != 1 {
		t.Fatalf("expected running refCount=1 after close, got running=%v refCount=%d", running, refCount)
	}

	if err := handle2.Close(); err != nil {
		t.Fatalf("Close() second error = %v", err)
	}
	running, refCount = supervisorState(supervisor)
	if running || refCount != 0 {
		t.Fatalf("expected stopped refCount=0 after close, got running=%v refCount=%d", running, refCount)
	}
}

func TestSupervisorStartPortConflict(t *testing.T) {
	addr := "127.0.0.1:0"
	cfg := config.DefaultResolvedConfig()
	cfg.Memory.Proxy.ListenAddr = addr

	supervisor := NewSupervisor()
	_, err := supervisor.Start(SupervisorOptions{
		ProviderID:  memprovider.ProviderCodex,
		ProjectRoot: t.TempDir(),
		Config:      &cfg,
		ListenFunc: func(network, addr string) (net.Listener, error) {
			return nil, syscall.EADDRINUSE
		},
	})
	if err == nil {
		t.Fatalf("expected port conflict error")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("expected port conflict error, got %q", err.Error())
	}
	running, refCount := supervisorState(supervisor)
	if running || refCount != 0 {
		t.Fatalf("expected no start after conflict, running=%v refCount=%d", running, refCount)
	}
}

func supervisorState(supervisor *Supervisor) (bool, int) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return supervisor.running, supervisor.refCount
}

type stubListener struct {
	addr   net.Addr
	closed chan struct{}
	once   sync.Once
}

func newStubListener(addr string) *stubListener {
	return &stubListener{
		addr:   stubAddr(addr),
		closed: make(chan struct{}),
	}
}

func (l *stubListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, http.ErrServerClosed
}

func (l *stubListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *stubListener) Addr() net.Addr {
	return l.addr
}

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }

func (a stubAddr) String() string { return string(a) }
