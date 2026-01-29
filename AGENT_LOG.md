# AGENT_LOG

## 2026-01-18 — Phase 1 implementation plan (initial)

- Phase 1 target per `specs/phase_1/PHASE_1.md`: planning-only agent integration (generate/refine/deps infer), durable plan file, validation/explainability, and a usable ready-task loop (list/show/pick + manual status updates).
- Keep dependencies low and the implementation clear (Go-first).
- Prefer a single, human-readable plan file stored at repo root; use JSON to avoid YAML dependencies.
- Agent runtime integration will be pluggable via an external command hook that returns machine-readable JSON (full plan or patch ops), with a manual-mode fallback.

## 2026-01-18 — Repo organization update

- Moved Phase 1 spec into `specs/phase_1/PHASE_1.md`.
- Added `specs/phase_1/IMPLEMENTATION_PLAN.md` capturing the Phase 1 build order and architecture.

## 2026-01-18 — Phase 1 milestone sub-specs

- Created one sub-spec per Phase 1 milestone under `specs/phase_1/milestones/` (M1–M6).
- Linked milestone docs from `specs/phase_1/IMPLEMENTATION_PLAN.md`.

## 2026-01-18 — M1: Foundation (Go module + Plan IO + Atomic Writes + Validate)

- Implemented a minimal Go CLI skeleton (`cmd/blackbird`) with `init` and `validate`.
- Added `internal/plan` with:
  - `WorkGraph` / `WorkItem` types (JSON, RFC3339 timestamps via `time.Time`).
  - Strict JSON loading (`DisallowUnknownFields`) and pretty-printed JSON saving.
  - Atomic write semantics (temp file in same dir + fsync file + rename + fsync dir).
  - Validation for required fields, reference existence, parent/child consistency, and hierarchy cycle detection.
- Documented the plan file schema and M1 validation behavior in `README.md`.
- Note: could not run `go test`/`gofmt` in this environment because the Go toolchain was not available (`go` not found). Run `go test ./...` locally to verify.

## 2026-01-18 — Repo initialized as git

- The project is now a git repo (no commits yet).
- No `origin` remote is configured yet. `go.mod` is set to `github.com/jbonatakis/blackbird` (update if/when the canonical remote URL differs).

## 2026-01-18 — Phase 1: when agent integration begins (milestones)

- Phase 1 agent integration starts at **M5** (agent request/response schema + external runtime adapter via `BLACKBIRD_AGENT_CMD`).
- The first user-visible, end-to-end agent-backed commands land in **M6** (`plan generate`, `plan refine`, `deps infer`).

## 2026-01-18 — M2: deps + readiness + list/show/set-status

- Added dependency DAG utilities in `internal/plan`:
  - reverse deps (`Dependents`)
  - unmet deps computation (`UnmetDeps`)
  - dependency cycle detection with a readable cycle path (`DepCycle`)
- Extended `Validate` to reject dependency cycles (deps must form a DAG).
- Implemented CLI commands:
  - `blackbird list` (default: actionable/ready leaf tasks)
  - `blackbird show <id>` (deps + dependents + readiness explanation + prompt)
  - `blackbird set-status <id> <status>` (updates status + `updatedAt`, writes atomically)
- Readiness semantics decision:
  - depsSatisfied means all deps are `done`
  - a task is considered actionable (READY in list) only when `status==todo` and deps are satisfied
  - `status==blocked` is treated as a manual override: even if deps are satisfied, it remains blocked until cleared

## 2026-01-18 — M3: manual edit commands (CRUD + hierarchy + dep editing)

- Added a runnable CLI entrypoint at `cmd/blackbird/main.go` (routes to `internal/cli.Run`).
- Implemented manual graph editing commands in `internal/cli`:
  - `add`, `edit <id>`, `delete <id>`, `move <id> --parent <parentId|root> [--index <n>]`
  - `deps add/remove/set`
- Added a small mutation layer in `internal/plan/mutate.go` to keep edits safe and consistent:
  - updates `updatedAt` on all touched nodes
  - rejects dependency cycles (typed error with a readable cycle path)
  - rejects hierarchy cycles on move (typed error with a readable cycle path)
  - delete safety semantics:
    - default refuses delete when node has children or external dependents
    - `--cascade-children` deletes subtree
    - `--force` removes dep edges from remaining nodes that depended on deleted nodes (keeps plan valid)
- Tightened validation: `depRationale` keys must reference existing IDs and also appear in `deps`.
- Added unit tests covering cycle prevention and delete safety (`internal/plan/mutate_test.go`).

## 2026-01-18 — Code review (Phase 1, M3)

- Reviewed manual edit CLI + mutation layer for M3 readiness.
- Noted issues around mutation side effects on failed dep edits and visibility of forced delete detachments.
- Flagged missing tests for CLI-level CRUD/deps flows.
- Logged findings in `specs/phase_1/CODE_REVIEW_M3.md`.

## 2026-01-18 — Validation review (Phase 1, M3 fixes)

- Reviewed user fixes for dep-edit rollback, delete output, and parent-cycle guard.
- Checked new tests in `internal/plan/mutate_test.go` and `internal/cli/manual_test.go`.
- Added a follow-up finding about duplicate detached IDs in forced deletes; logged in `specs/phase_1/code_reviews/CODE_REVIEW_M3.md`.
- Fixed `DeleteItem` to dedupe detached IDs; added test coverage.
- Re-checked `internal/plan/mutate.go` and `internal/cli/manual.go` to verify the findings match current code.
- Fixed dep edit rollback to restore prior `updatedAt` on cycle errors.
- Added tests covering `updatedAt` stability on failed dep edits.
- `delete --force` now reports detached dependency IDs; added CLI test coverage.
- Added parent-cycle guard in `parentCycleIfMove` with a test for invalid parent loops.

## 2026-01-18 — Validation review (Phase 1, M3 follow-up)

- Reviewed dedupe fix for forced delete detached IDs and the new test case.

## 2026-01-18 — M4: pick interactive loop

- Added `blackbird pick` command with a simple numbered selection loop and action prompts.
- Default selection matches list readiness (ready leaf tasks); supports `--include-non-leaf`, `--all`, and `--blocked`.
- Shows task details via existing `show` output and allows status transitions to `in_progress`, `done`, or `blocked`.
- Prints an explanatory message when no ready tasks are available, with guidance to use `list --blocked` or `show`.
- Added CLI tests for `pick` covering status updates and empty-state messaging.
- Switched prompt input to a shared reader to avoid buffered stdin loss in tests.

## 2026-01-18 — Spec update (Phase 1, patch ops alignment)

- Aligned patch operation names in `specs/phase_1/PHASE_1.md` with M5 (add/update/delete/move/set_deps/add_dep/remove_dep).

## 2026-01-18 — M5: agent schema + runtime adapter

- Added `internal/agent` types for request/response, patch ops, and validation.
- Implemented JSON extraction (single object or fenced ```json) with strict errors.
- Added external runtime adapter with provider selection, timeouts, retries, and stderr capture.
- Documented agent runtime configuration and JSON I/O rules in `README.md`.

## 2026-01-18 — M6: agent-backed planning flows

- Added CLI flows for `plan generate`, `plan refine`, and `deps infer` with interactive prompts, validation, and summaries (provider/model included).
- Implemented clarification Q&A loop for agent responses (bounded retries).
- Added patch application helper and plan diff summary to support refine/deps infer outputs.
- Updated README with M6 command guidance and planning flow notes.
- Tweaked agent prompts to show progress and to reprompt on invalid choices.

## 2026-01-18 — Repo review (Phase 1 status)

- Reviewed Phase 1 implementation against specs and milestones; noted a few risks in agent patch handling and clarification flow behavior.
- Ran `go test ./...` (all packages passed).

## 2026-01-18 — Agent debug logging tweak

- Moved agent request debug logging into the runtime so every attempt logs when `BLACKBIRD_AGENT_DEBUG=1` is set.

## 2026-01-18 — Default system prompt for plan requests

- Added a default `systemPrompt` to plan generate/refine/deps infer requests to enforce strict JSON responses and schema rules.

## 2026-01-18 — Claude JSON schema support

- Added `jsonSchema` request metadata and wired it for plan flows; runtime passes `--json-schema` when provider is Claude.

## 2026-01-18 — Plan generate preview

- Show the full plan tree before prompting accept/revise/no in `plan generate`.

## 2026-01-18 — Prompt guidance to avoid meta planning tasks

- Updated the default plan system prompt to discourage generic root placeholders and meta “design/plan” tasks.

## 2026-01-18 — Agent stdout/stderr streaming option

- Added `BLACKBIRD_AGENT_STREAM=1` to stream agent stdout/stderr live while still capturing output for JSON extraction.

## 2026-01-28 — Phase 2 execution dispatch spec

- Drafted a product-level spec for autonomous task execution dispatch, including goals, requirements, and definition of done in `specs/phase_2/EXECUTION_DISPATCH.md`.

## 2026-01-23 — Agent default timeout adjustment

- Increased `internal/agent` default runtime timeout to 10 minutes to avoid premature plan generation timeouts.

## 2026-01-23 — Agent progress indicator

- Added a simple progress indicator during agent runs so long operations show activity in the CLI.

## 2026-01-28 — Execution run types

- Added `internal/execution` package with RunRecord, RunStatus, and ContextPack types.
- Included task/dependency context structs and JSON tags (omitempty for optional fields).
- Added unit tests covering JSON round-trip and omission of optional fields.

## 2026-01-28 — Execution selector

- Added ReadyTasks selection logic in `internal/execution/selector.go` with deterministic ordering.
- Added unit tests covering readiness filtering and empty graph behavior.

## 2026-01-28 — Run record storage

- Added `SaveRun` with atomic write semantics and `.blackbird/runs/{task-id}/{run-id}.json` layout.
- Added storage tests for writing and basic validation errors.

## 2026-01-28 — Run record queries

- Added ListRuns, LoadRun, and GetLatestRun with sorted output and missing-dir handling.
- Added tests for listing order, missing data, and latest selection.

## 2026-01-28 — Runs history command

- Added `blackbird runs` command with optional `--verbose` output and table formatting.
- Added CLI tests for table output, verbose logs, and no-run message.

## 2026-01-28 — Question detection

- Added question parsing for AskUserQuestion tool output with JSON scanning.
- Added tests covering detection, no-questions, and error handling.

## 2026-01-28 — Execution context builder

- Added BuildContext to assemble task context, dependency summaries, and project snapshot.
- Loads snapshot from `.blackbird/snapshot.md` with fallbacks to `OVERVIEW.md`/`README.md`.
- Added tests for task/dependency inclusion and snapshot loading.

## 2026-01-28 — Execution lifecycle

- Added new plan statuses (queued, waiting_user, failed) and lifecycle transition validation.
- Implemented UpdateTaskStatus with atomic plan updates and state machine checks.
- Added tests for valid transitions and rejection of invalid transitions.

## 2026-01-28 — Agent launcher

- Added LaunchAgent to execute agent commands with context pack input and capture stdout/stderr.
- Detects AskUserQuestion output to switch to waiting_user status.
- Added tests for success, waiting_user detection, and failure exit codes.

## 2026-01-28 — Execute command

- Added `blackbird execute` loop to run ready tasks, update statuses, and store run records.
- Added CLI test for executing a single task and persisting run history.

## 2026-01-28 — Question resume

- Added ResumeWithAnswer to validate answers against parsed questions and build continuation context.
- Extended ContextPack to include questions and answers.
- Added tests for resume validation and invalid options.

## 2026-01-28 — Failure handling

- Added execute-loop test covering failure path and continued execution of subsequent tasks.
- Verified failed tasks are marked failed and run records persist per task.

## 2026-01-28 — Resume command

- Added `blackbird resume` to answer waiting_user questions and relaunch the agent.
- Added CLI test for resuming a waiting run and completing the task.

## 2026-01-28 — Retry command

- Added `blackbird retry` to reset failed tasks with failed run history back to todo.
- Added tests for retry success and missing failed run guard.

## 2026-01-28 — Agent execution bridge

- Added ExecuteTask wrapper to build context and launch the agent for a task.
- Added test coverage for ExecuteTask success path.

## 2026-01-28 — Parent task status updates

- Marked agent-exec, exec-dispatch, human-in-loop, run-records, safety-recovery, and exec-cli as done after completing their child work.

## 2026-01-28 — Execution docs and tests

- Documented execution commands and snapshot behavior in `README.md`.
- Added `internal/execution/README.md` with architecture overview.
- Marked exec-docs and exec-tests complete after expanding execution test coverage.

## 2026-01-28 — Execution file operations

- Added execution response schema parsing and file operation application.
- Updated launcher/execute/resume flows to require JSON responses and apply file ops.
- Updated tests and docs to reflect execution output contract.

## 2026-01-28 — Execution uses native agent edits

- Removed JSON file-op execution contract; agents now edit the working tree directly.
- Launcher no longer parses file ops; execute/resume just record stdout/stderr.
- Updated tests and docs to reflect native agent execution.

## 2026-01-28 — Execution auto-approve flags

- Added provider-specific auto-approve flags for headless execution runs.
- Codex uses `exec --full-auto`; Claude uses `--permission-mode acceptEdits`.

## 2026-01-28 — Claude permission mode update

- Updated Claude auto-approve flag to `--permission-mode dontAsk` to cover command execution prompts.

## 2026-01-28 — Execution system prompt

- Added a system prompt in execution context authorizing non-destructive commands and file edits without confirmation.

## 2026-01-28 — Claude permission mode bypass

- Switched Claude auto-approve flag to `--permission-mode bypassPermissions` for execution runs.

## 2026-01-29 — TUI action key handling scaffold

- Added `internal/tui/model.go` with `ActionMode` tracking and Update() handling for action keys (g/r/e/s), including ready-task guard for execute and pending status change state.
- Added `internal/tui/action_wrappers.go` with Bubble Tea commands that wrap CLI actions and capture stdout/stderr into a message.
- Added Bubble Tea dependency to `go.mod`.
- `go test ./...` failed locally because the Bubble Tea module could not be fetched (no network), leaving `go.sum` without entries.

## 2026-01-29 — Phase 3: TUI Dashboard

- Chose Bubble Tea for the TUI: Go-native, low-dependency, and well-suited for terminal UI patterns.
- Pane layout: left tree pane for task navigation, right pane for task/run detail and execution info, bottom bar for status/help.
- Navigation design: vim-style keys for movement, tab-style switching between panes.
- CLI integration: zero-args routing in `cli.Run` to launch the TUI as the default entrypoint when no command is provided.
- Execution dashboard: reads run records to populate active/previous runs and uses a live timer for elapsed time display.
- Action integration: TUI actions wrap existing CLI flows (execute, resume, retry, status updates) via command wrappers to reuse logic.
- Risks: blocking execution while wrapping CLI commands and terminal sizing issues; mitigated by running actions in Bubble Tea commands and handling `WindowSizeMsg` updates for layout resizing.
- Deviations: none noted from the Phase 3 plan.

## 2026-01-29 — TUI bottom bar

- Added bottom bar renderer with action hints, ready/blocked counts, and inverted styling via lipgloss.
- Wired action-in-progress spinner state and action names into the TUI model with a tick-based spinner.
- Updated TUI view to include the bottom bar and added lipgloss dependency to go.mod.

## 2026-01-29 — TUI action wrappers

- Expanded `internal/tui/action_wrappers.go` with plan/execute/resume/set-status commands returning typed Bubble Tea messages.
- Captured CLI stdout/stderr for TUI actions and added success flags to completion messages.
- Updated the TUI model to handle new action completion message types.

## 2026-01-29 — TUI detail pane renderer

- Added `internal/tui/detail_view.go` with `RenderDetailView` to format selected item details, dependencies, dependents, readiness, and prompt using lipgloss.
- Added viewport clipping for tall content and a minimal empty-selection fallback.
- Added Bubble Tea `bubbles/viewport` dependency in `go.mod`.
- Added `internal/tui/detail_view_test.go` covering detail rendering and empty selection output.

## 2026-01-29 — TUI execution dashboard view

- Added `internal/tui/execution_view.go` to render the execution dashboard (active run status, elapsed time, log excerpts, and task summary) with lipgloss styling.
- Added deterministic elapsed-time formatting via an overridable time source.
- Added `internal/tui/execution_view_test.go` covering active-run rendering, log tailing, and empty state output.
- `go test ./internal/tui/...` failed locally due to Go build cache permission restrictions in this environment.

## 2026-01-29 — TUI run loader

- Added run data loader and periodic refresh for the TUI using execution run storage.
- Model now loads latest run records on init and after execute/resume, with missing `.blackbird/runs` handled gracefully.
- Added `internal/tui/run_loader_test.go` covering missing run data and latest-run selection per task.

## 2026-01-29 — Live timer tick for elapsed time

- Added `internal/tui/timer.go` with a 1-second Bubble Tea tick command and active-run detection helper.
- Wired timer scheduling into `internal/tui/model.go` so ticks only run while runs are active.
- Added `internal/tui/timer_test.go` covering active run detection.

## 2026-01-29 — TUI tree view renderer

- Added `internal/tui/tree_view.go` with hierarchical plan tree rendering, expand/collapse handling, selection highlight, and status/readiness styling via lipgloss.
- Introduced `plan.ReadinessLabel` for shared readiness labeling; updated CLI list/pick paths to use it.
- Extended TUI model to track `expandedItems` and `filterMode` defaults for upcoming navigation/filter work.

## 2026-01-29 — TUI keyboard navigation + detail scrolling

- Added keyboard navigation handling in `internal/tui/model.go` for tree movement (up/down, j/k, home/end), expand/collapse (enter/space), pane toggle (tab), and filter cycling (f).
- Implemented visible-item traversal helpers and parent detection to keep selection aligned with render order and filter state.
- Added detail pane paging state and applied `pgup/pgdown` scrolling via the viewport offset in `internal/tui/detail_view.go`.
- Added unit tests for visible navigation, filter behavior, and selection snapping in `internal/tui/model_test.go`.
- `go test ./internal/tui/...` failed due to Go build cache permissions in this environment (`operation not permitted`).

## 2026-01-29 — TUI base model tests

- Added basic TUI model tests covering quit command handling, window size updates, and placeholder view text (`internal/tui/model_basic_test.go`).
- `go test ./internal/tui/...` failed locally due to Go build cache permission restrictions (`operation not permitted` while opening a cache file).

## 2026-01-29 — TUI entrypoint wiring

- Updated `cli.Run` to launch the TUI when no args are provided.
- Added `internal/tui/start.go` to load/validate the plan, create the Bubble Tea program, and run it with an alt screen.
- Switched TUI action wrappers to invoke the `blackbird` binary via `os/exec` to avoid a `cli` ↔ `tui` import cycle.
- `go test ./...` failed locally due to missing `go.sum` entries for Bubble Tea-related modules in this environment.

## 2026-01-29 — TUI scaffold verification

- Verified `internal/tui` package, Bubble Tea model implementation, and `tui.Start()` entrypoint wiring are already present.
- Confirmed `cli.Run` routes zero-arg invocation to the TUI and `go.mod` includes Bubble Tea dependencies.
- No code changes required for the requested scaffold task.

## 2026-01-29 — TUI pane layout + view rendering

- Implemented two-column tree/detail layout in `internal/tui/model.go` with lipgloss borders, active-pane highlighting, and size-aware pane splitting.
- Wired view rendering to use `RenderTreeView` and `RenderDetailView`, keeping the bottom bar and status prompt overlay.
- Added a unit test to ensure the main view renders both tree and detail content (`internal/tui/model_view_test.go`).

## 2026-01-28 — TUI Implementation: Comprehensive Testing and Documentation

### Implementation Approach

The TUI implementation uses Bubble Tea as the terminal UI framework with a split-pane design:

**Architecture decisions:**
- **Framework choice: Bubble Tea** - Go-native, well-suited for terminal patterns, low-dependency footprint
- **Pane layout**: Left pane shows hierarchical task tree (with expand/collapse), right pane shows task details or execution dashboard
- **Tab modes**: `t` key switches between Details view (task info, deps, readiness, prompt) and Execution view (active run status, elapsed time, logs, task summary)
- **Navigation model**: Vim-style keys (`j/k`, `up/down`, `home/end`) for tree navigation, `tab` to switch active pane, `enter/space` to expand/collapse parent tasks
- **Filter system**: `f` key cycles through FilterModeAll → FilterModeReady → FilterModeBlocked to show relevant tasks
- **Action integration**: Wraps existing CLI commands (execute, resume, set-status, plan generate/refine) via Bubble Tea commands to reuse validation and execution logic
- **State management**: Model tracks selected task, expanded items, filter mode, active pane, action state (in-progress with spinner), and run data (loaded from `.blackbird/runs/`)

**Key design decisions:**
1. **Tree rendering with visibility tracking**: `visibleItemIDs()` computes which items are shown based on parent expansion state and current filter, enabling correct navigation and selection snapping
2. **Elapsed time display**: Uses overridable `timeNow` function for testability, formats durations as `HH:MM:SS` with live 1-second tick updates when runs are active
3. **Action spinner integration**: When actions run (execute, generate, refine, etc.), model shows a spinner in the bottom bar with descriptive action text
4. **Viewport scrolling**: Detail pane supports `pgup/pgdown` scrolling for tall content via offset tracking
5. **Zero-args entry**: `cli.Run([])` routes to `tui.Start()`, making TUI the default interactive mode

**Risks encountered and mitigations:**
- **Risk**: Blocking execution during CLI command wrapping → **Mitigation**: All actions run as Bubble Tea commands (async) with completion messages that update the model
- **Risk**: Terminal sizing issues → **Mitigation**: Handle `tea.WindowSizeMsg` to resize panes dynamically, with minimum width constraints in `splitPaneWidths()`
- **Risk**: Navigation desyncing from tree visibility → **Mitigation**: `ensureSelectionVisible()` snaps selection to first visible item when filter changes hide current selection
- **Risk**: Circular import between `cli` and `tui` → **Mitigation**: TUI actions invoke `blackbird` binary via `os/exec` instead of direct function calls

### Testing Strategy

Added comprehensive unit tests covering core TUI logic without requiring full Bubble Tea program execution:

**Test files created:**
1. `internal/tui/tree_view_test.go` - Tree rendering logic tests:
   - Empty plan handling
   - Single item rendering
   - Parent-child hierarchy display
   - Collapsed parent behavior (children hidden)
   - Filter matching logic (FilterModeAll, FilterModeReady, FilterModeBlocked)
   - Root ID detection with orphaned nodes
   - Expansion state tracking

2. `internal/tui/model_test.go` - Navigation and state management tests:
   - `nextVisibleItem()` / `prevVisibleItem()` with boundary conditions (stay at start/end)
   - Navigation with collapsed parents (skips hidden children)
   - `visibleItemIDs()` with filter modes
   - `toggleExpanded()` state transitions
   - `ensureSelectionVisible()` when filter hides current selection
   - `isParent()` detection
   - `nextFilterMode()` cycling
   - `splitPaneWidths()` calculations with various window sizes
   - `detailPageSize()` with different window heights

3. `internal/tui/timer_test.go` - Elapsed time calculation tests:
   - Zero duration
   - Various durations (seconds, minutes, hours)
   - Completed runs (using completedAt timestamp)
   - Edge cases (end before start, millisecond truncation)
   - Time mocking for deterministic tests

4. `internal/cli/cli_test.go` - CLI TUI integration tests:
   - `Run([])` without plan file returns "plan file not found" error
   - `Run(["help"])` displays usage information
   - `Run(["init"])` creates valid plan file
   - `Run(["validate"])` checks plan validity
   - Documented that full TUI launch test is skipped (requires TTY)

**Test coverage highlights:**
- Tree rendering with various plan structures (empty, single item, hierarchies, collapsed states)
- Navigation helpers respect expanded/collapsed state and filters
- Elapsed time formatting handles all duration ranges and edge cases
- CLI routing to TUI verified (zero-args behavior)
- All core logic paths tested without mocking Bubble Tea internals

**Design rationale for testability:**
- Extracted pure functions (`formatElapsed`, `filterMatch`, `rootIDs`, `isExpanded`) for unit testing
- Used overridable time source (`timeNow`) for deterministic elapsed time tests
- Separated visibility computation (`visibleItemIDs`, `visibleBranch`) from rendering
- Navigation helpers (`nextVisibleItem`, `prevVisibleItem`) operate on model state without UI dependencies

All tests pass locally and provide coverage for critical TUI logic paths without requiring interactive terminal sessions.

## 2026-01-29 — README TUI update

- Documented the TUI default entrypoint (`blackbird`) and key bindings in `README.md`.
- Noted the execution selection behavior (ready tasks include non-leaf items).

## 2026-01-28 — Documentation cleanup and ignore rules

- Rewrote `README.md` with a public-facing overview, install steps, command summary, and configuration details.
- Added `docs/README.md` as a documentation index linking to workflows, specs, and testing notes.
- Expanded `.gitignore` to cover `.blackbird/` run data, coverage output, and test binaries.

## 2026-01-28 — Testing docs reorganization

- Moved testing markdown files into `docs/testing/` and updated cross-references.

## 2026-01-28 — Notes and bugs doc locations

- Moved `AGENT_QUESTIONS_IMPLEMENTATION.md` into `docs/notes/`.
- Moved `BUGS_AND_FIXES.md` into `docs/testing/`.

## 2026-01-29 — TUI plan refresh tick

- Added plan reload in the TUI every 5 seconds to keep task statuses in sync during execution.
- Plan updates now refresh on action completion alongside run data.

## 2026-01-28 — Repository code review

- Reviewed core plan, execution, agent, CLI, and TUI modules for correctness.
- Flagged status validation/schema mismatches, question ID validation gaps, and LaunchAgent error handling behavior.
- Created `BACKLOG.md` capturing missing features vs `OVERVIEW.md`.

## 2026-01-29 — Codex plan flow parity

- Aligned agent runtime provider args with execution behavior so plan flows use non-interactive codex/claude flags.
- Added coverage for provider arg prefixing in `internal/agent/runtime_test.go`.
