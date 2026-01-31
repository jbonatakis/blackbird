BINARY := blackbird
CMD := ./cmd/blackbird

.PHONY: build

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
