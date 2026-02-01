# AGENT_LOG

## 2026-02-01 — @path picker integration (plan modals)

- Integrated @-triggered file picker across plan generate/refine modals (rendering, key routing, insertion, ESC handling).
- Added integration/unit coverage for picker open/query/insert and modal rendering/ESC behavior.
- Tests: `GOCACHE=/tmp/blackbird-go-cache go test ./internal/tui/...`.

## 2026-02-01 — Plan modal ESC guard in model update

- Added model-level ESC handling for plan generate/refine that closes the @ file picker first and only closes the modal when the picker is not open.

## 2026-02-01 — Plan refine picker key routing

- Routed plan-refine key handling to prioritize the @ file picker when open.
- Added model-level tests for plan-refine picker open-on-@ and enter insertion behavior.

## 2026-02-01 — Plan generate picker key routing

- Routed plan-generate key handling through the @ file picker, with anchor tracking, query updates, and insert/cancel behavior tied to the focused field.
- Ensured ESC closes the picker without dismissing the modal; tab/shift+tab close the picker without moving focus.
- Added plan-generate picker routing tests (open/query/backspace, tab close, enter insert) plus modal ESC behavior coverage.
- `go test ./internal/tui/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — Plan generate picker state

- Added file picker state + per-field anchors to `PlanGenerateForm` with helpers to open/close/apply selections.
- Added helpers for applying picker selections to textareas/textinput with cursor positioning.
- Added unit tests covering picker open tracking and selection insertion behavior.

## 2026-02-01 — Hard/soft deps spec: soft deps under-the-hood only

- Updated `specs/improvements/HARD_SOFT_DEPS_AND_UNBLOCKS_MOST.md`: soft deps are not rendered or editable in CLI/TUI; they are plan-only (visible in plan JSON or in code). Added "Display and editing: soft deps are under-the-hood only" section, clarified Dependents display shows only hard dependents, updated Non-goals and Done criteria.

## 2026-02-01 — Hard/soft deps and unblocks-most spec

- Added `specs/improvements/HARD_SOFT_DEPS_AND_UNBLOCKS_MOST.md`: spec for two dependency lists (hard `deps`, soft `softDeps`) with mutual exclusivity per dependent; readiness uses only hard deps; ready-task ordering by "unblocks most" (prefer task that the most other not-done tasks depend on, hard or soft), then task ID tie-break. Covers schema, validation, readiness, selector order, Dependents/depRationale/cycle detection, and backward compatibility.

## 2026-02-01 — TUI file picker state

- Added `internal/tui/file_picker_state.go` with `FilePickerState`, anchor metadata, and helpers (open/close/reset, selection clamping).
- Added unit tests for picker state selection behavior and anchor span in `internal/tui/file_picker_state_test.go`.
- `go test ./internal/tui/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — File lookup (@path) spec

- Added `specs/improvements/FILE_LOOKUP_AT_PATH.md`: spec for @-triggered file picker in plan generate and plan refine text boxes. Typing `@` opens a picker that filters files by path prefix; Enter inserts chosen path, Escape cancels. Covers scope (which fields), UX (filtering, keys), technical approach (file listing, picker state, key routing, insert), touchpoints, edge cases, and out-of-scope items (.gitignore, CLI, other modals).

## 2026-02-01 — TUI Change agent shortcut [c] and position

- Changed agent shortcut from [a] to [c] (Change agent) to differentiate from [g] Generate plan.
- Moved "Change agent" to bottom of home actions, just above Quit, in home view and bottom bar.
- Updated model key handler, home_view, bottom_bar, trim priorities, and tests; `go test ./internal/tui/...` passes.

## 2026-02-01 — Global config spec

- Added `specs/improvements/GLOBAL_CONFIG.md`: spec for global configuration with `~/.blackbird/config.json` (global) and `<project>/.blackbird/config.json` (project overrides), precedence project > global > built-in, initial keys for TUI run/plan refresh intervals.

## 2026-02-01 — Execution launcher default agent selection

- Defaulted execution launcher to use selected agent when runtime provider is unset.
- Added execution launcher tests for selected-agent defaulting and explicit provider preservation.
- `go test ./internal/execution/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — TUI bottom bar agent label

- Added shared agent label helper and rendered active agent in bottom bar for Home/Main with compact counts when space is tight.
- Trimmed low-priority main-view action hints to keep agent + status indicators visible.
- Added/updated bottom bar tests; ran `go test ./...`.

## 2026-02-01 — TUI agent label tests

- Added model update coverage for agent selection save and explicit bottom bar agent-label assertions for Home/Main.

## 2026-02-01 — Plan agent consolidation log

- Logged shared plan-agent consolidation (CLI/TUI parity, shared response/status helpers) per work item.

## 2026-02-01 — Agent selection load

- Added agent selection loader with defaults, config-path helper, and validation errors.
- Added tests for missing/valid/invalid selection configs and default agent helper.

## 2026-02-01 — TUI agent selection

- Added Home-screen agent selection modal with keybinding, persisted selection save/load, and UI status display.
- Wired agent selection loading into TUI init and refreshes in-memory state on save.
- Updated runtime default provider selection to respect saved agent config when env vars are unset.
- Added tests for agent selection modal, save command, home hints, and runtime selection; ran `go test ./...`.

## 2026-02-01 — Agent registry

- Added a small agent registry with stable IDs for Claude/Codex plus lookup helpers and tests.
- Wired runtime defaults and provider arg selection to use the registry IDs.

## 2026-02-01 — Parity changes (TUI + shared helpers)

- Removed remaining TUI subprocess usage for plan actions and set-status, keeping everything in-process and aligned with CLI behavior.
- Consolidated shared plan status/response helpers so CLI and TUI use the same mutation and agent-response paths.

## 2026-02-01 — TUI set-status in-process

- Added shared plan helpers `ParseStatus` and `SetStatus` to reuse status mutation logic across CLI/TUI, preserving updatedAt updates and parent completion propagation.
- Switched TUI set-status to in-process plan mutation + `SaveAtomic` (no subprocess), and updated CLI to use the shared helpers.
- Added tests for plan status helpers and TUI set-status command behavior (parent propagation + timestamp updates).
- Ran `go test ./...`; failure persists in `internal/agent` due to `RequestPlanPatch` undefined in `internal/agent/response_test.go` (pre-existing).

## 2026-02-01 — Agent response helper

- Added `agent.ResponseToPlan` shared helper to convert agent responses into plans with full-plan timestamp normalization and patch application.
- Updated CLI/TUI plan flows to use the shared helper and removed duplicated response conversion logic; adjusted related tests.
- Verified CLI/TUI response handling already routes through `agent.ResponseToPlan`; no additional changes required for the wiring task.

## 2026-01-31 — README and docs rework

- Reworked README to be high-level only: what Blackbird is, install, quickstart, short TUI pointer, and a documentation table linking to `docs/`.
- Moved detailed content into `docs/`: COMMANDS.md (plan, manual edits, execution), TUI.md (layout and key bindings), READINESS.md, CONFIGURATION.md (agent env vars), FILES_AND_STORAGE.md.
- Updated docs/README.md as the documentation index with a Reference section for the new docs and consistent markdown links.

## 2026-01-31 — TUI live output append test

- Added a unit test to validate live output buffer appending and continued listening on live stream updates (`internal/tui/live_output_model_test.go`).

## 2026-01-31 — TUI streaming output cmd

- Added a live-output done message and updated the streaming Cmd/Update loop to stop cleanly when the channel closes.
- Added unit tests for live output command chunk delivery and channel-close behavior.

## 2026-01-31 — TUI plan view visual fixes

- **Top cut off (root cause)**: Lipgloss applies Height to the inner block then adds top+bottom border, so each pane is `availableHeight + 2` lines. Total = `(availableHeight+2) + 2` (newline + bar). Use `availableHeight = windowHeight - 5` so total = `windowHeight - 1`, staying under terminal height. Rendering exactly `windowHeight` lines can cause first-line redraw bugs in some terminals/bubbletea; keeping output one line short ensures the top border is visible.
- **Detail pane viewport**: `applyViewport` uses `model.windowHeight` (pane content height); `detailPageSize()` returns `windowHeight - 5` to match.
- **Plan pane top border short**: The title was inserted by replacing runes, which corrupted ANSI codes. Fixed by rebuilding the top border line; use first content line width as target and pad with middle dashes if short.
- **Details box off-screen**: Each pane's rendered width is content width + 2 (left and right border). splitPaneWidths used left+right+gap=total so left+right=total-1, making total rendered width (left+2)+(right+2)=total+3. Fixed by splitting so left+right=total-4; then (left+2)+(right+2)=total and both panes fit on screen.
- **Pane layout revert**: Reverted to the state when the top was fully visible: availableHeight = windowHeight-5, removed ensureContentHeight (no tree padding). Kept splitPaneWidths(total-4) and 1:3 split so both panes fit on screen. Bottom bar may jump when switching Details/Execution if pane heights differ.
- **Lipgloss/TUI learnings**: Added `docs/notes/LIPGLOSS_TUI_LEARNINGS.md` with layout rules (height/width + borders, top-border rebuild, JoinHorizontal, viewport, testing).
- **Jump on task change**: Removed reset of `detailOffset` when changing selection (up/down, j/k, home, end) so scroll position is preserved when moving between tasks. Tab switch and filter change still reset `detailOffset` to 0.
- Tests: Adjusted `TestDetailPageSize` for the new formula; `TestViewRendersPlaceholderText` now uses `windowHeight: 3` so at least one content line is shown (with windowHeight 2, availableHeight was 0 so only the bar was rendered).

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

## 2026-01-29 — TUI quit key fix

- Removed `q` as a global quit key in the TUI to avoid exiting during text entry.
- Updated bottom bar hints and docs to reflect `ctrl+c` as the quit shortcut.
- Adjusted TUI tests to match the new quit behavior.

## 2026-01-29 — Codex skip git repo check

- Added `--skip-git-repo-check` to codex provider args so plan flows run outside git repos.

## 2026-01-29 — TUI home screen spec

- Added a Phase 3 spec for a TUI Home screen and missing-plan handling in `specs/phase_3/TUI_HOME_SCREEN_PLAN.md`.

## 2026-01-30 — TUI model view mode + plan gating helpers

- Added `ViewMode` (Home/Main) and `planExists` fields to the TUI Model with default `ViewModeHome`.
- Added `hasPlan()`/`canExecute()` helpers and updated execute gating to use `canExecute()`.
- Added unit tests for plan existence + execution gating in `internal/tui/model_basic_test.go`.

## 2026-01-30 — TUI home view renderer

- Added `internal/tui/home_view.go` with `RenderHomeView` to render a centered home screen (title, tagline, plan status, action list) with muted/shortcut/action styling via lipgloss.
- Wired the home view into the main render path when `viewMode == ViewModeHome`.
- Added `internal/tui/home_view_test.go` covering home view output for missing and present plans.

## 2026-01-30 — TUI startup missing plan handling

- Updated TUI startup to initialize with an empty in-memory plan, planExists=false, and Home view.
- Adjusted plan loader to treat missing plan files as non-errors, returning an empty graph with planExists=false.
- Added plan loader coverage for missing plan files and planExists assertions.

## 2026-01-30 — TUI Home view integration tweaks

- Updated `Model.View()` to select the Home view at the top-level before modal overlays and kept split-pane rendering for ViewModeMain.
- Simplified bottom bar hints for the Home screen and hid status counts when no plan exists.
- Added tests for home bottom bar hints/count hiding and for home view rendering in `Model.View()`.
- `go test ./internal/tui/...` failed locally due to Go build cache permission restrictions (`operation not permitted`).

## 2026-01-30 — TUI home key routing + plan gating

- Added Home-screen key routing in `internal/tui/model.go` (g/v/r/e/ctrl+c, h toggles views with plan guard) while preserving Main view behavior.
- Set `planExists=true` and switched to Main view after successful plan save.
- Added unit tests covering Home key toggling, gated actions, and ctrl+c quit (`internal/tui/model_home_keys_test.go`).

## 2026-01-30 — TUI plan refresh missing-plan handling

- Updated plan refresh handling so PlanDataLoaded always applies the latest plan state, even on errors.
- Added plan load error surface via action output to show validation/load errors without crashing.
- Added unit test coverage for PlanDataLoaded error state updates.

## 2026-01-30 — TUI home validation error banner

- Added `planValidationErr` to the TUI model and propagated it through plan loading.
- Plan loader now stores a concise validation error summary (first validation error) when the plan file exists but is invalid, while keeping `planExists=true`.
- Home view renders a red (Color 196) bordered error banner with remediation guidance when a validation error is present.
- Updated tests for plan loading validation state and home banner rendering.
- `go test ./internal/tui/...` failed locally due to Go build cache permission restrictions.

## 2026-01-30 — TUI missing-plan and home gating tests

- Added `newStartupModel` helper and test to assert startup state with no plan file (planExists=false, Home view).
- Added missing-plan PlanDataLoaded update coverage to ensure planExists stays false without error output.
- Added execute gating test to ensure home execute action stays disabled when no ready tasks exist.

## 2026-01-30 — TUI home-screen test fixes

- **cli**: `TestRunZeroArgsWithoutPlanFile` now skips (TUI starts without plan file; Run would block in tui.Start()).
- **tui**: `TestViewRendersPlaceholderText` asserts home view "No plan found" for default empty-plan view.
- **tui**: `TestModelViewRendersTreeAndDetail` sets `viewMode: ViewModeMain` and `planExists: true` so main view (tree/detail) is rendered.
- **tui**: `TestLoadPlanData` fixture now includes `AcceptanceCriteria: []string{}` so validation passes.
- **tui**: `TestTabModeToggle` and `TestTabModeResetsDetailOffset` set `viewMode: ViewModeMain` so 't' key toggles tab.
- All tests pass: `go test ./...`

## 2026-01-31 — Shared execution runner API

- Added `internal/execution` runner API with `ExecuteConfig`/`ResumeConfig`, `RunExecute`, `RunResume`, and `ExecuteResult` stop reasons.
- Moved CLI execute/resume orchestration onto the shared runner with task-start/finish hooks for logging.
- Added helper for pulling questions from the latest waiting run and error helpers for waiting/no-question cases.
- Added runner unit tests covering execute completion, waiting-user stop, and resume success; verified `go test ./...` passes.

## 2026-01-31 — CLI runner integration touch-ups

- Updated `runResume` to build a resume context pack from the latest waiting run using `ListRuns` + `ResumeWithAnswer`, added SIGINT/SIGTERM context handling, and passed the prebuilt context into `execution.RunResume`.
- Extended `execution.ResumeConfig` to accept an optional prebuilt `ContextPack` and validate task ID alignment before resuming.

## 2026-01-31 — Runner tests, TUI in-process execution, and docs

- Expanded `internal/execution/runner_test.go` with table-driven coverage for stop reasons, ready-loop ordering, status updates, and context cancellation in execute/resume.
- Updated TUI execute/resume actions to run in-process via the shared runner with cancellable contexts; quit/ctrl+c now invokes the cancel func.
- Added TUI tests for action completion/cancel behavior plus in-process ExecuteCmd/ResumeCmd integration coverage.
- Documented the in-process execution model in `README.md` and marked `specs/improvements/IN_PROCESS_EXECUTIONS.md` complete.
- Marked `runner-tests-and-docs` and `tui-runner-integration` as done in `blackbird.plan.json`.

## 2026-01-31 — Plan timestamp normalizer helper

- Added `plan.NormalizeWorkGraphTimestamps` to normalize all work-item `createdAt`/`updatedAt` values using a single provided time.
- Wired CLI/TUI plan response handling to normalize full-plan responses using the shared helper and pass a single timestamp through patch application.
- Added plan normalization unit test to ensure timestamps update without mutating other fields.
- `go test ./internal/plan/...` failed locally due to Go build cache permission restrictions.

## 2026-01-31 — Full-plan timestamp normalization follow-up test

- Added execution lifecycle coverage ensuring status updates succeed after normalized full-plan timestamps (`internal/execution/lifecycle_test.go`).
- `go test ./internal/execution/...` failed locally due to Go build cache permission restrictions (`operation not permitted` in Go build cache).

## 2026-01-31 — Plan normalization wiring verification

- Verified `responseToPlan` in `internal/cli/agent_helpers.go` and `internal/tui/action_wrappers.go` already normalizes full-plan responses via `plan.NormalizeWorkGraphTimestamps` with caller-provided `now` and leaves patch application unchanged; no code changes required.

## 2026-01-31 — Plan normalization tests (full-plan responses)

- Added CLI and TUI tests to ensure full-plan agent responses normalize createdAt/updatedAt to a single now and pass plan.Validate.
- CLI test builds a parent/child plan to validate normalization across items; TUI test covers single-item full-plan response.
- Ran `go test ./...` (all packages passed).

## 2026-01-31 — Leaf-only readiness

- Updated ReadyTasks to exclude non-leaf items (childIds non-empty), so only leaf todo tasks with satisfied deps are executable.
- Expanded ReadyTasks test coverage to include a parent/child case and assert parent containers are excluded.

## 2026-01-31 — ReadyTasks leaf-only verification

- Verified `internal/execution/selector.go` already skips items with non-empty `ChildIDs` and documents leaf-only readiness.
- Confirmed ReadyTasks tests include a parent/child case to ensure containers are excluded.

## 2026-01-31 — ReadyTasks leaf-only tests

- Added `TestReadyTasksLeafOnly` in `internal/execution/selector_test.go` to assert non-leaf todo tasks (with `childIds`) are excluded even when deps are satisfied, while leaf tasks with satisfied deps are returned.

## 2026-01-31 — Parent completion propagation

- When a task is set to done, parents are now auto-marked done when all of their children are done (and recursion up the hierarchy). This unblocks tasks that depend on parent containers (e.g. a top-level "testing" task that depends on "chess-core" and "cli-interface").
- Added `plan.PropagateParentCompletion(g, childID, now)` in `internal/plan/parent.go`; called from `execution.UpdateTaskStatus` and `cli.runSetStatus` when status is set to done.
- Tests: `plan/parent_test.go` (no parent, parent not all children done, parent all children done, grandparent chain); `execution/lifecycle_test.go` TestUpdateTaskStatusPropagatesParentCompletion.
- Ran `go test ./...` (all packages passed).

## 2026-01-31 — Execution streaming hooks

- Added optional stdout/stderr streaming writers to `execution.ExecuteConfig` and `execution.ResumeConfig`, wired through the runner to the agent launcher.
- Introduced `LaunchAgentWithStream` and `StreamConfig` to support per-run live output sinks while preserving existing `LaunchAgent` behavior.
- Added launcher test coverage to ensure provided stream writers receive output.

## 2026-01-31 — Execution runner stream tee

- Updated execution launcher stream wiring so stdout/stderr are always copied to capture buffers and any streaming sink (plus env-based streaming when enabled).
- Extended launcher streaming test to assert captured stdout remains populated when streaming is active.

## 2026-01-31 — TUI live execution buffers

- Added live stdout/stderr buffers to the TUI model with streaming listeners for execute/resume, plus safe reset on completion.
- Wired in-process execution/resume to stream output into the TUI via `StreamStdout`/`StreamStderr` writers.
- Execution view now shows live output when no active run record exists, without changing completed-run rendering.
- Tests: added `TestRenderExecutionViewLiveOutput`; ran `go test ./...`.

## 2026-01-31 — Execution view live buffers routing

- Updated execution view log output selection to prefer live buffers during execute/resume actions and fall back to run record output otherwise.
- Added tests to cover live output overriding run logs and run logs used when not in progress.
- Ran `go test ./...` (all packages passed).

## 2026-01-31 — TUI execution tab guard tweak

- Allowed `t` tab toggling during execute/resume actions while keeping the guard for other in-progress actions.
- Added tab toggle tests for execute/resume and updated the in-progress guard test to use a non-exec action name.

## 2026-01-31 — TUI tab toggle test consolidation

- Consolidated execute/resume tab-toggle coverage into a table-driven test to assert `t` switches tabs during action-in-progress execute/resume states in `internal/tui/tab_mode_test.go`.
- Tests: `go test ./internal/tui/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — Shared agent response helper tests

- Added focused unit coverage for `agent.ResponseToPlan` in `internal/agent/response_test.go`, covering full-plan timestamp normalization and patch-application path.

## 2026-02-01 — TUI refine in-process

- Replaced TUI refine action subprocess call with an in-process agent request, adding a change-request modal and refine continuation handling for agent questions.
- Added a plan-refine modal, pending refine request tracking, and save helper to preserve view behavior while persisting refined plans.
- Tests: `go test ./...` failed in `internal/agent` (undefined `RequestPlanPatch` in `internal/agent/response_test.go`).

## 2026-02-01 — TUI subprocess cleanup

- Removed unused TUI subprocess wrappers for plan generate/refine and the os/exec command runner in `internal/tui/action_wrappers.go`.
- Updated CLI/TUI parity notes to reflect in-process plan refine and set-status behavior.

## 2026-02-01 — Test updates for shared response helper and TUI refine

- Fixed agent response helper test to use plan_refine request type.
- Added in-process plan refine TUI test using a stubbed agent response.

## 2026-02-01 — Shared plan defaults

- Added shared plan defaults in `internal/agent` for the plan system prompt, JSON schema, max question rounds, and max generate revisions.
- Updated CLI/TUI plan flows (including plan review modal) to use the shared defaults and constants, removing duplicated helpers.

## 2026-02-01 — Shared plan path helper

- Added `plan.PlanPath()` helper to compute the plan path from the current working directory and default filename.
- Updated CLI and TUI code to use the shared helper and added coverage for the helper.

## 2026-02-01 — TUI plan path helper wiring

- Updated TUI plan loader to use plan.PlanPath() instead of duplicating working-directory path logic.
- Adjusted TUI tests to use the shared plan path helper for plan file setup (plan loader, action wrappers, set-status).

## 2026-02-01 — Plan defaults test coverage

- Added unit tests for shared plan defaults (constants, JSON schema, system prompt) in `internal/agent/plan_defaults_test.go`.
- Fixed CLI execute/resume imports for shared plan helper and normalized plan path test to handle macOS tempdir symlinks.

## 2026-02-01 — Agent selection persistence

- Added atomic save helper for agent selection config under `.blackbird/agent.json`, including directory creation and schema serialization.
- Added tests covering save/load round-trip and invalid agent selection handling.

## 2026-02-01 — Agent selection config tests

- Added invalid-config fallback coverage for agent selection loading (missing field, unsupported schema, trailing data).

## 2026-02-01 — Home view agent display

- Added home view test coverage to ensure the selected agent label is rendered in the status area.

## 2026-02-01 — Documented agent selection

- Documented the Home view agent picker key and noted that the selection persists to `.blackbird/agent.json` in `docs/TUI.md`.

## 2026-02-01 — Plan flow agent selection

- Added agent metadata helper to default request provider from the active runtime while preserving explicit overrides.
- Wired CLI/TUI plan generate/refine/deps infer flows to apply the runtime provider to plan request metadata.
- Added unit tests covering provider defaults vs explicit metadata overrides.

## 2026-02-01 — Plan task tree builder

- Added `plan.BuildTaskTree` to derive ordered parent/child hierarchy from parentId references, with stable sibling ordering (childIds order + sorted remainder) and missing-parent roots.
- Wired TUI tree rendering/visibility and plan review modal to use the shared tree structure; CLI tree listing and feature roots now use the shared tree roots/children.
- Added plan-level tests for tree ordering, root handling, and missing-parent behavior; updated TUI tests for parent detection to include parentId.
- `go test ./internal/plan/... ./internal/tui/... ./internal/cli/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — TUI tree lipgloss renderer

- Switched TUI tree rendering to use lipgloss tree renderer for branch/indent formatting.
- Preserved task line data (id, status, readiness label, title) with existing color styles and expansion indicator.
- Kept filter/expand behavior by building tree nodes conditionally and omitting collapsed children.

## 2026-02-01 — Compact tree line format

- Simplified TUI tree lines to compact readiness abbreviations and removed redundant status column.
- Added truncation helpers for IDs/titles based on pane width to keep lines readable in narrow terminals.

## 2026-02-01 — TUI file picker file listing

- Added workspace file listing helper for the @ file picker that walks cwd, skips .git/.blackbird, enforces a max result cap, and returns forward-slashed relative paths.
- Added unit tests covering prefix filtering, noise-dir skipping, forward-slash normalization, and max-result limit handling.
- `go test ./internal/tui/...` failed due to Go build cache permission restrictions (operation not permitted).

## 2026-02-01 — File picker filtering helper

- Added deterministic file picker filtering helper that normalizes slashes, filters by prefix, and sorts/limits results.
- Updated workspace file listing to return ordered matches and added unit tests for filtering behavior (ordering, empty query, slash normalization).

## 2026-02-01 — File picker insertion helper

- Added a helper to replace the @query span with the selected path and return the updated value plus cursor rune index.
- Added unit tests covering single-line and multi-line replacements.
- `go test ./internal/tui/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — File picker key handling helper

- Added file picker key routing helper with actions (none/insert/cancel), query/match updates, and selection movement, plus matching utilities.
- Added unit tests covering open-on-@ behavior, selection moves, enter/esc/tab handling, and query edits.
- `go test ./internal/tui/...` failed due to Go build cache permission restrictions (`operation not permitted`).

## 2026-02-01 — TUI file picker rendering

- Added a file picker list renderer using lipgloss with selection highlight, empty-state message, and fixed sizing for modal use.
- Added tests covering closed-state rendering, empty-state message sizing, and selection window output.

## 2026-02-01 — File picker table-driven tests

- Added table-driven tests for file picker listing/filtering and key handling actions/bounds in `internal/tui/file_picker_test.go`.

## 2026-02-01 — Granularity file picker support

- Added file picker tests for granularity textinput, covering open/query updates, tab cancellation, and enter insertion.
- Normalized backslash paths in file picker filtering/listing for cross-platform match behavior.
- Ran `GOCACHE=/tmp/blackbird-go-cache go test ./internal/tui/...`.

## 2026-02-01 — Plan generate picker rendering

- Rendered the file picker list inside the plan generate modal, aligned to the active field and clamped to modal width/height.
- Added a render test ensuring picker output appears between the description and constraints sections when open.

## 2026-02-01 — Plan generate modal picker integration tests

- Added plan generate modal integration tests covering @-open, enter insertion in description/constraints, and tab/shift+tab focus changes.
- Updated file picker tab handling to close and allow focus movement, and adjusted form-level picker tests accordingly.
- Ran `GOCACHE=/tmp/blackbird-go-cache go test ./internal/tui/...`.

## 2026-02-01 — Plan refine picker state

- Added file picker state + anchor tracking to `PlanRefineForm` with open/close/apply helpers and key routing.
- Rendered the file picker list inside the plan refine modal and aligned ESC handling so it closes the picker before the modal.
- Added tests for refine picker open/query/insertion, modal rendering, and ESC behavior; ran `GOCACHE=/tmp/blackbird-go-cache go test ./internal/tui/...`.

## 2026-02-01 — Plan refine picker render verification

- Verified `RenderPlanRefineModal` already renders the file picker list when open, with width clamped to the textarea/modal content for alignment.
- No code changes needed for the picker render task.

## 2026-02-01 — Documented @path lookup in TUI

- Added TUI docs note describing @ file lookup behavior in plan generate/refine text areas and key controls.

## 2026-02-01 — Plan refine picker modal tests

- Added plan-refine modal integration tests for file picker ESC close and tab/shift+tab focus changes.
- Ran `GOCACHE=/tmp/blackbird-go-cache go test ./internal/tui/...`.
