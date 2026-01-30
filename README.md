# Blackbird

Go-first CLI for durable, dependency-aware planning and execution with AI agents.

## What it does

- Maintains a validated, dependency-aware work graph in a single JSON plan file.
- Surfaces readiness so you can see what is actionable next.
- Runs agent-backed plan generation/refinement and dependency inference.
- Executes ready tasks with a headless agent runtime, logging runs for traceability.
- Provides a TUI for interactive navigation, detail views, and execution status.

## Install (from source)

Requires Go 1.22+.

- Build a local binary:
  - `go build -o blackbird ./cmd/blackbird`
- Or install into `GOBIN`:
  - `go install ./cmd/blackbird`

## Quickstart

- Initialize a plan file:
  - `blackbird init`
- Generate an initial plan with the agent:
  - `blackbird plan generate`
- Launch the TUI (default entrypoint):
  - `blackbird`
- List ready work:
  - `blackbird list`
- Execute ready tasks:
  - `blackbird execute`
- Resume a waiting task:
  - `blackbird resume <taskID>`
- View run history:
  - `blackbird runs <taskID>`

The plan file lives at repo root as `blackbird.plan.json`.

## TUI overview

Running `blackbird` with no arguments launches the TUI. CLI commands like
`blackbird plan`, `blackbird execute`, and `blackbird list` are unchanged.

Layout:

- Left pane: plan tree with status and readiness labels
- Right pane: details or execution dashboard (toggle with `t`)
- Bottom bar: action shortcuts and ready/blocked counts

Key bindings:

- `up/down` or `j/k`: move selection in the tree
- `enter` or space: expand/collapse parent items
- `tab`: switch focus between tree and detail panes
- `f`: cycle filters (all, ready, blocked)
- `pgup/pgdown`: scroll the detail pane
- `t`: switch details/execution tab
- `g`: plan generate
- `r`: plan refine
- `e`: execute ready tasks
- `u`: resume waiting task (when available)
- `s`: set status for selected item
- `ctrl+c`: quit

## Core commands

Plan and graph management:

- `blackbird plan generate` generates a plan from a project description.
- `blackbird plan refine` applies agent-proposed edits to the current plan.
- `blackbird deps infer` proposes dependency updates with rationale.
- `blackbird validate` checks plan integrity and dependency consistency.
- `blackbird show <id>` prints task details and readiness explanations.
- `blackbird set-status <id> <status>` updates task status manually.

Manual graph edits:

- `blackbird add --title "..." [--parent <parentId|root>]`
- `blackbird edit <id> --title "..." --description "..." --prompt "..."`
- `blackbird move <id> --parent <parentId|root> [--index <n>]`
- `blackbird delete <id> [--cascade-children] [--force]`
- `blackbird deps add <id> <depId>`
- `blackbird deps remove <id> <depId>`
- `blackbird deps set <id> [<depId> ...]`

Execution:

- `blackbird execute` runs ready tasks in dependency order.
- `blackbird runs <taskID>` lists runs for a task (`--verbose` shows logs).
- `blackbird resume <taskID>` answers questions and continues a waiting task.
- `blackbird retry <taskID>` resets failed tasks with failed runs back to `todo`.

## Readiness rules

- Deps are satisfied when **all deps have status `done`**.
- A task is actionable when **status is `todo`** and deps are satisfied.
- `blocked` is a manual override even if deps are satisfied.

## Agent runtime configuration

Blackbird invokes an external agent command for plan generation/refinement and
execution. Configuration is environment-based:

- `BLACKBIRD_AGENT_PROVIDER=claude|codex` selects the default command (defaults to `claude`).
- `BLACKBIRD_AGENT_CMD` overrides the command entirely (runs via `sh -c`).
- `BLACKBIRD_AGENT_STREAM=1` streams agent stdout/stderr live to the terminal.
- `BLACKBIRD_AGENT_DEBUG=1` prints the JSON request payload for debugging.

The command must emit exactly one JSON object on stdout (either the full stdout
or inside a fenced ```json block). Multiple objects or missing JSON fail fast.

## Files and storage

- Plan file: `blackbird.plan.json`
- Run records: `.blackbird/runs/<taskID>/<runID>.json`
- Optional snapshot file: `.blackbird/snapshot.md`
  - Fallbacks to `OVERVIEW.md`, then `README.md` if missing.

## Documentation

- Documentation index: `docs/README.md`
- Project overview: `OVERVIEW.md`
- Execution architecture: `internal/execution/README.md`
- Agent question flow: `docs/AGENT_QUESTIONS_FLOW.md`
- Plan review flow: `docs/PLAN_REVIEW_FLOW.md`
- Testing quickstart: `docs/testing/TESTING_QUICKSTART.md`
- Specs and milestones: `specs/`
