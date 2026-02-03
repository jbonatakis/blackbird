package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/memory"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

const defaultShutdownTimeout = 5 * time.Second

type SupervisorOptions struct {
	ProviderID      string
	ProjectRoot     string
	Config          *config.ResolvedConfig
	ShutdownTimeout time.Duration
	ListenFunc      func(network, addr string) (net.Listener, error)
}

type SupervisorHandle struct {
	once sync.Once
	stop func() error
}

func (h *SupervisorHandle) Close() error {
	if h == nil || h.stop == nil {
		return nil
	}
	var err error
	h.once.Do(func() {
		err = h.stop()
	})
	return err
}

type Supervisor struct {
	mu              sync.Mutex
	running         bool
	refCount        int
	server          *http.Server
	proxy           *Proxy
	listener        net.Listener
	addr            string
	shutdownTimeout time.Duration
}

func NewSupervisor() *Supervisor {
	return &Supervisor{}
}

var defaultSupervisor = NewSupervisor()

func StartSupervisor(opts SupervisorOptions) (*SupervisorHandle, error) {
	return defaultSupervisor.Start(opts)
}

func (s *Supervisor) Start(opts SupervisorOptions) (*SupervisorHandle, error) {
	if s == nil {
		return noopHandle(), nil
	}

	adapter := memprovider.Select(opts.ProviderID)
	if adapter == nil {
		return noopHandle(), nil
	}

	projectRoot := resolveProjectRoot(opts.ProjectRoot)
	cfg := resolveProxyConfig(opts.Config, projectRoot)
	if !adapter.Enabled(cfg.Memory) {
		return noopHandle(), nil
	}

	listenAddr := resolveListenAddr(cfg.Memory.Proxy.ListenAddr)
	tracePath := memory.TraceWALPath(projectRoot, "")
	traceOptions := trace.Options{
		MaxSizeBytes: int64(cfg.Memory.Retention.TraceMaxSizeMB) * 1024 * 1024,
		Retention:    time.Duration(cfg.Memory.Retention.TraceRetentionDays) * 24 * time.Hour,
		PrivacyMode:  !cfg.Memory.Proxy.Lossless,
	}

	s.mu.Lock()
	if s.running {
		s.refCount++
		handle := s.handleLocked()
		s.mu.Unlock()
		return handle, nil
	}

	listen := opts.ListenFunc
	if listen == nil {
		listen = net.Listen
	}
	listener, err := listen("tcp", listenAddr)
	if err != nil {
		s.mu.Unlock()
		return nil, formatListenError(listenAddr, err)
	}

	proxy, err := New(Config{
		Adapter:        adapter,
		APIBaseURL:     cfg.Memory.Proxy.UpstreamURL,
		ChatGPTBaseURL: cfg.Memory.Proxy.ChatGPTUpstreamURL,
		BaseURLPrefix:  adapter.BaseURLPrefix(),
		TracePath:      tracePath,
		TraceOptions:   traceOptions,
	})
	if err != nil {
		_ = listener.Close()
		s.mu.Unlock()
		return nil, fmt.Errorf("start memory proxy: %w", err)
	}

	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout
	}

	server := &http.Server{Handler: proxy}
	s.running = true
	s.refCount = 1
	s.server = server
	s.proxy = proxy
	s.listener = listener
	s.addr = listener.Addr().String()
	s.shutdownTimeout = shutdownTimeout

	handle := s.handleLocked()
	s.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.markStopped(server, proxy, listener)
		}
	}()

	return handle, nil
}

func (s *Supervisor) handleLocked() *SupervisorHandle {
	return &SupervisorHandle{stop: s.stop}
}

func (s *Supervisor) stop() error {
	if s == nil {
		return nil
	}

	var (
		server   *http.Server
		proxy    *Proxy
		listener net.Listener
		timeout  time.Duration
	)

	s.mu.Lock()
	if s.refCount == 0 {
		s.mu.Unlock()
		return nil
	}
	s.refCount--
	if s.refCount > 0 {
		s.mu.Unlock()
		return nil
	}

	server = s.server
	proxy = s.proxy
	listener = s.listener
	timeout = s.shutdownTimeout

	s.running = false
	s.server = nil
	s.proxy = nil
	s.listener = nil
	s.addr = ""
	s.shutdownTimeout = 0
	s.mu.Unlock()

	return shutdownProxy(server, proxy, listener, timeout)
}

func (s *Supervisor) markStopped(server *http.Server, proxy *Proxy, listener net.Listener) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.server != server {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.refCount = 0
	s.server = nil
	s.proxy = nil
	s.listener = nil
	s.addr = ""
	s.shutdownTimeout = 0
	s.mu.Unlock()

	_ = shutdownProxy(server, proxy, listener, defaultShutdownTimeout)
}

func shutdownProxy(server *http.Server, proxy *Proxy, listener net.Listener, timeout time.Duration) error {
	var err error
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}

	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		shutdownErr := server.Shutdown(ctx)
		cancel()
		if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			closeErr := server.Close()
			err = errors.Join(err, shutdownErr, closeErr)
		} else if shutdownErr != nil {
			err = errors.Join(err, shutdownErr)
		}
	}

	if listener != nil {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}

	if proxy != nil {
		if closeErr := proxy.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}

	return err
}

func resolveProxyConfig(cfg *config.ResolvedConfig, projectRoot string) config.ResolvedConfig {
	if cfg != nil {
		return *cfg
	}
	resolved, err := config.LoadConfig(projectRoot)
	if err != nil {
		return config.DefaultResolvedConfig()
	}
	return resolved
}

func resolveProjectRoot(projectRoot string) string {
	trimmed := strings.TrimSpace(projectRoot)
	if trimmed != "" {
		return trimmed
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func resolveListenAddr(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return config.DefaultMemoryProxyListenAddr
	}
	return trimmed
}

func formatListenError(addr string, err error) error {
	if isAddrInUse(err) {
		return fmt.Errorf("memory proxy listen address %s already in use", addr)
	}
	return fmt.Errorf("start memory proxy listener on %s: %w", addr, err)
}

func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return errors.Is(opErr.Err, syscall.EADDRINUSE)
	}
	return false
}

func noopHandle() *SupervisorHandle {
	return &SupervisorHandle{stop: func() error { return nil }}
}
