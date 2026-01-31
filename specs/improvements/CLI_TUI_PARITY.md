# CLI / TUI parity and shared code (DRY)

**status:** incomplete

## Purpose

Ensure **parity** between the CLI and TUI so that:

1. **Behavior is uniform** — The same operations (plan generate, refine, set-status, execute, resume, etc.) behave the same regardless of whether the user uses `blackbird` in a terminal or the TUI. No divergent semantics, validation, or error handling between access methods.

2. **Code is shared where possible (DRY)** — Logic for plan/agent flows, execution, and plan mutations lives in **shared packages** (e.g. `internal/plan`, `internal/agent`, `internal/execution`). CLI and TUI are thin entrypoints that call the same APIs instead of reimplementing or forking behavior.

3. **Future work stays aligned** — New features and bug fixes are implemented once in shared code; both CLI and TUI benefit. Duplication is the exception, not the norm.

## Principles

- **Single code path per capability** — For any given capability (e.g. “convert agent response to plan”, “run execute loop”, “update task status”), there should be one implementation that both CLI and TUI call. Prefer adding or using a function in `internal/plan`, `internal/agent`, or `internal/execution` over duplicating logic in `internal/cli` and `internal/tui`.

- **CLI and TUI are consumers, not owners** — CLI and TUI handle I/O (flags, prompts, TUI rendering, subprocess vs in-process) and call into shared packages for business logic. They should not each implement “response to plan”, “execute loop”, or “validate and save plan” in parallel.

- **Same rules, same validation** — Validation, normalization, and persistence rules (e.g. timestamp normalization for full-plan responses, `updatedAt >= createdAt`, atomic writes) are defined once and used by both. No “CLI does X, TUI does Y” for the same concept.

- **Uniform experience** — Users who switch between `blackbird` CLI and TUI should see consistent outcomes: same plan file shape, same status transitions, same execution/resume behavior. Differences should be limited to presentation (text vs interactive UI), not semantics.

## Current state

### Already shared (good)

- **Plan model, IO, validation** — `internal/plan`: types, Load/SaveAtomic, Validate, Clone, NormalizeWorkGraphTimestamps, mutate helpers. Both CLI and TUI use these.
- **Agent types and patch application** — `internal/agent`: Request/Response, ApplyPatch, schema. Both use the same types and patch semantics.
- **Execute / resume orchestration** — Per IN_PROCESS_EXECUTIONS: `internal/execution` provides RunExecute / RunResume; CLI and TUI both call them. Single code path for execute and resume.

### Still duplicated (to reduce)

- **responseToPlan** — “Convert agent response (full plan or patch) to a WorkGraph” exists in two places:
  - `internal/cli/agent_helpers.go` — used by CLI plan generate/refine/deps infer.
  - `internal/tui/action_wrappers.go` — used by TUI plan generate and plan review modal.
  - Both do the same thing: if full plan → `plan.NormalizeWorkGraphTimestamps`; if patch → `plan.Clone` + `agent.ApplyPatch`; else error. Error type differs (CLI uses `errors.New`, TUI uses `agent.RuntimeError`). Consolidating into a single function (e.g. in `internal/agent`) would remove duplication and keep behavior identical.

- **Other candidates** — As more features are added (e.g. set-status, plan refine, deps infer from TUI), prefer implementing or reusing shared helpers rather than reimplementing in TUI via subprocess or copy-paste. Each new flow should ask: “can both CLI and TUI call the same function?”

## Target state

- **responseToPlan** (or equivalent) lives in one place (e.g. `internal/agent`). Signature: same inputs (base plan, response, now) and outputs (WorkGraph, error). CLI and TUI both call it. Error type can be a shared type (e.g. `agent.ResponseError` or plain `error`) so both handle the same cases.

- **New features** — Any new “plan/agent” or “execution” capability is implemented in a shared package first; CLI and TUI are wired to that API. Duplication is only acceptable when there is a strong reason (e.g. TUI-specific prompt flow); even then, the core logic (e.g. “merge response into plan”) should be shared.

- **Documentation** — This doc and AGENT_LOG (or similar) reference the parity principle so that new work defaults to “add to shared package, call from both CLI and TUI.”

## Scope and non-goals

- **In scope:** Establishing the principle, documenting current duplication (e.g. responseToPlan), and guiding consolidation (e.g. move responseToPlan to internal/agent and have both call it). Optional: a short checklist for “new feature: did we add it in shared code and use it from both?”
- **Out of scope:** Changing UX or feature set; rewriting all existing code in one PR; forcing every tiny helper into a shared package (only substantive, user-affecting behavior). TUI-specific UI logic (rendering, key bindings, modals) stays in `internal/tui`.

## Success criteria

- At least the single largest duplication (responseToPlan) is removed: one implementation, both CLI and TUI call it.
- New plan/agent or execution features are implemented in shared code and invoked from both CLI and TUI unless explicitly justified otherwise.
- Validation and normalization behavior (timestamps, status, etc.) are identical for CLI and TUI; no “works in CLI, fails in TUI” (or vice versa) for the same operation.
