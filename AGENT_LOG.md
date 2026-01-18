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
