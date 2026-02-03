package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/memory"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
	memproxy "github.com/jbonatakis/blackbird/internal/memory/proxy"
	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

func main() {
	var providerID string
	var projectRoot string

	flag.StringVar(&providerID, "provider", memprovider.ProviderCodex, "memory provider id")
	flag.StringVar(&projectRoot, "project-root", "", "project root (default: cwd)")
	flag.Parse()

	root := strings.TrimSpace(projectRoot)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("resolve project root: %v", err)
		}
		root = wd
	}

	cfg, err := config.LoadConfig(root)
	if err != nil {
		log.Printf("load config failed, using defaults: %v", err)
		cfg = config.DefaultResolvedConfig()
	}

	adapter := memprovider.Select(providerID)
	if adapter == nil {
		log.Fatalf("unknown provider %q", providerID)
	}

	tracePath := memory.TraceWALPath(root, "")
	traceOptions := trace.Options{
		MaxSizeBytes: int64(cfg.Memory.Retention.TraceMaxSizeMB) * 1024 * 1024,
		Retention:    time.Duration(cfg.Memory.Retention.TraceRetentionDays) * 24 * time.Hour,
		PrivacyMode:  !cfg.Memory.Proxy.Lossless,
	}

	proxy, err := memproxy.New(memproxy.Config{
		Adapter:        adapter,
		APIBaseURL:     cfg.Memory.Proxy.UpstreamURL,
		ChatGPTBaseURL: cfg.Memory.Proxy.ChatGPTUpstreamURL,
		BaseURLPrefix:  adapter.BaseURLPrefix(),
		TracePath:      tracePath,
		TraceOptions:   traceOptions,
	})
	if err != nil {
		log.Fatalf("start proxy: %v", err)
	}
	defer func() { _ = proxy.Close() }()

	addr := strings.TrimSpace(cfg.Memory.Proxy.ListenAddr)
	if addr == "" {
		addr = config.DefaultMemoryProxyListenAddr
	}

	log.Printf("blackbird memory proxy listening on %s", addr)
	if err := http.ListenAndServe(addr, proxy); err != nil {
		log.Fatalf("proxy stopped: %v", err)
	}
}
