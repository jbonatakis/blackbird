BINARY := blackbird
CMD := ./cmd/blackbird

.PHONY: build test format tag

# Bump version: make tag BUMP=patch|minor|major
tag:
	@if [ -z "$(BUMP)" ]; then \
		echo "Usage: make tag BUMP=patch|minor|major"; exit 1; fi; \
	if [ "$(BUMP)" != "patch" ] && [ "$(BUMP)" != "minor" ] && [ "$(BUMP)" != "major" ]; then \
		echo "BUMP must be one of: patch, minor, major"; exit 1; fi; \
	latest=$$(git describe --tags --abbrev=0 2>/dev/null); \
	if [ -z "$$latest" ]; then \
		latest="v0.0.0"; fi; \
	v=$$(echo "$$latest" | sed 's/^v//'); \
	maj=$$(echo "$$v" | cut -d. -f1); min=$$(echo "$$v" | cut -d. -f2); pat=$$(echo "$$v" | cut -d. -f3); \
	case "$(BUMP)" in \
		major) maj=$$((maj+1)); min=0; pat=0;; \
		minor) min=$$((min+1)); pat=0;; \
		patch) pat=$$((pat+1));; \
	esac; \
	newtag="v$$maj.$$min.$$pat"; \
	git tag -a "$$newtag" -m "Release $$newtag" && echo "Created tag $$newtag. Push with: git push --tags"

build:
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	suffix=$$(echo "$$branch" | sed 's/[^a-zA-Z0-9_-]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$$//'); \
	if [ "$$branch" = "main" ]; then \
		go build -o $(HOME)/.local/bin/$(BINARY) $(CMD); \
		echo "Built $(BINARY) for main branch"; \
	else \
		go build -o $(HOME)/.local/bin/$(BINARY)-$$suffix $(CMD); \
		echo "Built $(BINARY)-$$suffix for $$branch branch"; \
	fi

test:
	go test ./...

format:
	go fmt ./...
