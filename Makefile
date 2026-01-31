BINARY := blackbird
CMD := ./cmd/blackbird

.PHONY: build

build:
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" = "main" ]; then \
		go build -o $(HOME)/.local/bin/$(BINARY) $(CMD); \
		echo "Built $(BINARY) for main branch"; \
	else \
		go build -o $(HOME)/.local/bin/$(BINARY)-$$branch $(CMD); \
		echo "Built $(BINARY)-$$branch for $$branch branch"; \
	fi
