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

## CLI → TUI action mapping

For each CLI command, how the TUI implements (or does not implement) the same operation. “Implementation” = how the TUI runs the logic (subprocess vs in-process / shared code). “Parity notes” = gaps or differences.

| CLI command                      | TUI trigger / equivalent            | Implementation | Parity notes                                                                                                                                                                                                                                                                                                                              |
| -------------------------------- | ----------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **init**                         | (none)                              | —              | TUI has no “init” action; plan is loaded on start. Creating a plan from scratch is done via [g] Generate.                                                                                                                                                                                                                                 |
| **validate**                     | (none)                              | —              | No standalone “validate” in TUI. Plan is validated on load; `planValidationErr` is shown when invalid.                                                                                                                                                                                                                                    |
| **list**                         | Main view (tree + filters)          | In-memory      | TUI shows tree/list with filter mode ([f]); conceptually same as `list` / `list --tree`.                                                                                                                                                                                                                                                  |
| **pick**                         | (none)                              | —              | No direct “pick” in TUI. User selects a task in the tree, then [s] set-status or [e] execute.                                                                                                                                                                                                                                             |
| **plan generate**                | [g] from home                       | **In-process** | TUI uses `GeneratePlanInMemory` (agent in-process); form collects description/constraints/granularity; then plan review modal (accept/revise/reject). Revise uses `RefinePlanInMemory`. Does **not** call CLI subprocess.                                                                                                                 |
| **plan refine**                  | [r] from home or main               | **In-process** | TUI collects a change request via modal and runs `RefinePlanInMemory`, matching the CLI refine behavior without subprocesses.                                                                                                                                                                                                              |
| **show \<id\>**                  | Detail pane (selected item)         | In-memory      | Selecting an item in the tree shows the same kind of info as `show <id>`.                                                                                                                                                                                                                                                                 |
| **set-status \<id\> \<status\>** | [s] with selection → SetStatusModal | **In-process** | TUI uses shared plan mutation helpers (`plan.SetStatus` + `SaveAtomic`), matching CLI semantics without subprocesses.                                                                                                                                                                                                                     |
| **add**                          | (none)                              | —              | No TUI for adding items. CLI-only.                                                                                                                                                                                                                                                                                                        |
| **edit \<id\>**                  | (none)                              | —              | No TUI for editing items. CLI-only.                                                                                                                                                                                                                                                                                                       |
| **delete \<id\>**                | (none)                              | —              | No TUI for deleting items. CLI-only.                                                                                                                                                                                                                                                                                                      |
| **move \<id\>**                  | (none)                              | —              | No TUI for moving items. CLI-only.                                                                                                                                                                                                                                                                                                        |
| **deps add/remove/set**          | (none)                              | —              | No TUI for manual dep edits. CLI-only.                                                                                                                                                                                                                                                                                                    |
| **deps infer**                   | (none)                              | —              | No TUI for deps infer. CLI-only.                                                                                                                                                                                                                                                                                                          |
| **runs \<taskID\>**              | Execution tab / run data            | In-memory      | TUI loads run data via `RunDataRefreshCmd` and shows it in the execution pane.                                                                                                                                                                                                                                                            |
| **execute**                      | [e]                                 | **In-process** | TUI uses `ExecuteCmd` → `execution.RunExecute` (same as CLI). Single code path.                                                                                                                                                                                                                                                           |
| **resume \<taskID\>**            | [u] when `CanResume`                | **In-process** | TUI uses `ResumeCmd` → `execution.RunResume`; answers collected via AgentQuestionForm. Same code path as CLI.                                                                                                                                                                                                                             |
| **retry \<taskID\>**             | (none)                              | —              | No TUI for retry. CLI-only.                                                                                                                                                                                                                                                                                                               |

### Summary

- **In-process (shared code):** plan generate (TUI form → `GeneratePlanInMemory` / `RefinePlanInMemory`), plan refine, set-status, execute, resume.
- **CLI-only (no TUI):** init, validate, pick, add, edit, delete, move, deps add/remove/set, deps infer, retry.
- **Parity gaps:** None for plan refine/set-status; remaining gaps are feature-availability only (CLI-only commands).

## Current state

### Already shared (good)

- **Plan model, IO, validation** — `internal/plan`: types, Load/SaveAtomic, Validate, Clone, NormalizeWorkGraphTimestamps, mutate helpers. Both CLI and TUI use these.
- **Agent types and patch application** — `internal/agent`: Request/Response, ApplyPatch, schema. Both use the same types and patch semantics.
- **Execute / resume orchestration** — Per IN_PROCESS_EXECUTIONS: `internal/execution` provides RunExecute / RunResume; CLI and TUI both call them. Single code path for execute and resume.

### Still duplicated (to reduce)

- **responseToPlan** — consolidated into `agent.ResponseToPlan` and used by both CLI/TUI plan flows; keep new plan-related behavior wired through the shared helper to avoid drift.

- **Other candidates** — See the [CLI → TUI action mapping](#cli--tui-action-mapping) above. Plan refine from TUI currently uses a subprocess and never collects the change request; it should be wired like plan review “Revise” (modal for input, then shared in-process refine). set-status in TUI uses subprocess; it could call shared plan mutation + SaveAtomic instead. As more features are added (e.g. deps infer from TUI), prefer implementing or reusing shared helpers. Each new flow should ask: “can both CLI and TUI call the same function?”

## Target state

- **responseToPlan** (or equivalent) lives in one place (e.g. `internal/agent`). Signature: same inputs (base plan, response, now) and outputs (WorkGraph, error). CLI and TUI both call it. Error type can be a shared type (e.g. `agent.ResponseError` or plain `error`) so both handle the same cases.

- **New features** — Any new “plan/agent” or “execution” capability is implemented in a shared package first; CLI and TUI are wired to that API. Duplication is only acceptable when there is a strong reason (e.g. TUI-specific prompt flow); even then, the core logic (e.g. “merge response into plan”) should be shared.

- **Documentation** — This doc and AGENT_LOG (or similar) reference the parity principle so that new work defaults to “add to shared package, call from both CLI and TUI.”

## Scope and non-goals

- **In scope:** Establishing the principle, documenting current duplication (e.g. responseToPlan), and guiding consolidation (e.g. move responseToPlan to internal/agent and have both call it). Optional: a short checklist for “new feature: did we add it in shared code and use it from both?”
- **Out of scope:** Implementing new features from the CLI in the TUI; Changing UX or feature set; rewriting all existing code in one PR; forcing every tiny helper into a shared package (only substantive, user-affecting behavior). TUI-specific UI logic (rendering, key bindings, modals) stays in `internal/tui`.

## Success criteria

- At least the single largest duplication (responseToPlan) is removed: one implementation, both CLI and TUI call it.
- New plan/agent or execution features are implemented in shared code and invoked from both CLI and TUI unless explicitly justified otherwise.
- Validation and normalization behavior (timestamps, status, etc.) are identical for CLI and TUI; no “works in CLI, fails in TUI” (or vice versa) for the same operation.
