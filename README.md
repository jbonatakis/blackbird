# Blackbird

[![CI](https://github.com/jbonatakis/blackbird/actions/workflows/ci.yml/badge.svg)](https://github.com/jbonatakis/blackbird/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jbonatakis/blackbird)](https://github.com/jbonatakis/blackbird/releases)

The control plane for durable, dependency-aware planning and execution with AI agents.

<img src="static/blackbird-execution.gif" width="600" />

## What it does

- Builds a dependency-aware work graph based on the spec you provide.
- Spawns headless instances of your selected coding agent to execute tasks in order. It uses your existing settings, respects CLAUDE.md/AGENTS.md, and never runs out of context.
- Supports Claude Code and Codex. More to come.
- Allows you to approve, reject, or revise the agent's work after every task (or not -- you're in control).
- Streams agent output as it works.

## Install

### Homebrew
```
brew tap jbonatakis/tap
brew install blackbird
```

### From Source
Requires Go 1.25+.

```bash
go build -o blackbird ./cmd/blackbird
# or: make build
```

## Quickstart

Just open a repo and run `blackbird`

## Documentation

| Topic                                        | Doc                                                                      |
| -------------------------------------------- | ------------------------------------------------------------------------ |
| **Commands** (plan, manual edits, execution) | [docs/COMMANDS.md](docs/COMMANDS.md)                                     |
| **TUI** (layout, key bindings)               | [docs/TUI.md](docs/TUI.md)                                               |
| **Readiness rules**                          | [docs/READINESS.md](docs/READINESS.md)                                   |
| **Agent configuration**                      | [docs/CONFIGURATION.md](docs/CONFIGURATION.md)                           |
| **Files and storage**                        | [docs/FILES_AND_STORAGE.md](docs/FILES_AND_STORAGE.md)                   |
| **Documentation index**                      | [docs/README.md](docs/README.md)                                         |
| **Project overview**                         | [OVERVIEW.md](OVERVIEW.md)                                               |
| **Execution architecture**                   | [internal/execution/README.md](internal/execution/README.md)             |
| **Agent question flow**                      | [docs/AGENT_QUESTIONS_FLOW.md](docs/AGENT_QUESTIONS_FLOW.md)             |
| **Plan review flow**                         | [docs/PLAN_REVIEW_FLOW.md](docs/PLAN_REVIEW_FLOW.md)                     |
| **Testing**                                  | [docs/testing/TESTING_QUICKSTART.md](docs/testing/TESTING_QUICKSTART.md) |
| **Specs and milestones**                     | [specs/](specs/)                                                         |
