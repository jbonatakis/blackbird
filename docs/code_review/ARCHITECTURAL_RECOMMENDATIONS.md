# Architectural Recommendations

High-level recommendations to improve structure, testability, and maintainability without changing product behavior. Ordered by impact vs. effort.

---

## 1. Explicit config / working directory (medium impact, low effort)

**Current:** `plan.PlanPath()` uses `os.Getwd()`; run storage base dir is derived from the plan path in multiple places. Everything assumes “current working directory.”

**Issue:** Tests and future features (e.g. multiple plans, alternate roots) must rely on `os.Chdir` or duplicated path logic. There’s no single place that defines “where the plan lives” and “where runs live.”

**Recommendation:** Introduce a small config that is built once at process start and passed through:

- **Plan path** – e.g. from env `BLACKBIRD_PLAN` or default `./blackbird.plan.json` relative to cwd.
- **Base dir** – directory containing the plan (used for `.blackbird/runs`, snapshot, etc.).
- **Runtime** – optional; can stay env-based for now.

CLI and TUI would build this config once (e.g. in `main` and in `tui.Start`) and pass it into `RunExecute`, `RunResume`, plan load/save, and run query functions. Tests can inject a temp dir without changing global cwd.

**Scope:** Add a `Config` (or `Env`) struct; thread it through `execution.ExecuteConfig` / `ResumeConfig` and any plan load/save call sites that today call `plan.PlanPath()`. Plan path can remain defaulted via a helper that uses cwd when config doesn’t override it.

---

## 2. Single “operations” layer for CLI and TUI (high impact, medium effort)

**Current:** Execute, resume, plan generate, and plan refine are implemented twice: once in `internal/cli` (e.g. `execute.go`, `resume.go`, `agent_flows.go`) and once in `internal/tui` (e.g. `action_wrappers.go`, plan modals). Both do: load plan, call agent/execution, save plan, handle errors.

**Issue:** Any change to “what execute does” or “how we save after refine” must be done in two places. Parity bugs (e.g. status handling, stream env) are easy to introduce.

**Recommendation:** Introduce an **operations** (or **app**) layer that owns the core workflows and is IO-agnostic:

- **Execute** – load plan, run execution loop, persist runs and plan updates; return result/error.
- **Resume** – load plan, run resume, persist; return result/error.
- **PlanGenerate** – call agent, normalize/validate, optionally save; return plan + questions or error.
- **PlanRefine** – same idea for refine.

CLI would: parse args, build config, call the operation, then write to stdout/stderr. TUI would: call the same operations from `tea.Cmd` and map results to `tea.Msg`. Shared helpers like `ParseStatus`, `SetStatus`, `ResponseToPlan` stay; the *orchestration* (load → call agent/execution → save) lives in one place.

**Scope:** New package e.g. `internal/ops` or `internal/app` with functions that take config + minimal args (paths, runtime, stream writers, etc.) and return structured results. Migrate `cli` and `tui` to call these instead of duplicating flow. Optionally keep thin CLI handlers that only do flag parsing and IO.

---

## 3. Split the TUI Model (medium impact, medium effort)

**Current:** `internal/tui/model.go` is a single large `Model` (1000+ lines) holding plan, run data, all view state, all action modes, and all form state.

**Issue:** Hard to navigate, test, or change one view/mode without touching the rest. Bubble Tea doesn’t require a single model, only that `Update` returns a `Model`.

**Recommendation:** Split by responsibility while keeping one top-level `Model` that delegates:

- **Option A – by view:** e.g. `HomeModel` (plan load, create/open) and `MainModel` (tree, detail, execution, modals). Top-level `Model` holds current view and delegates `Update`/`View` to the active one.
- **Option B – by domain:** e.g. `PlanState` (plan, selectedID, filterMode), `RunState` (runData, live output), `UIState` (window size, activePane, tabMode, actionMode). Top-level `Model` composes these and owns view logic.

Either way, modals (plan generate, plan refine, agent question, etc.) can stay as separate types that receive size/plan and return a `tea.Msg` on completion. The goal is to shrink the single `Update` switch and make each sub-model testable on its own.

---

## 4. Run store and plan IO behind interfaces (low impact short-term, useful for tests and evolution)

**Current:** Execution and CLI/TUI call `plan.Load`, `plan.SaveAtomic`, `execution.SaveRun`, `execution.ListRuns`, etc. directly. All are file-based.

**Recommendation:** Introduce small interfaces so that “load/save plan” and “list/save runs” are abstracted:

- **PlanStore** – e.g. `Load() (WorkGraph, error)`, `Save(WorkGraph) error` (path can be part of the impl or config).
- **RunStore** – e.g. `SaveRun(record) error`, `ListRuns(taskID) ([]RunRecord, error)`, `GetLatestRun(taskID) (*RunRecord, error)`.

Default implementations remain the current file-based code. Tests can use in-memory or temp-dir implementations. Execution and the (future) operations layer take stores as dependencies instead of calling package-level functions. This also makes the “explicit config” recommendation cleaner: config can hold store implementations.

**Scope:** Start with RunStore only if you want minimal change; execution already has a clear storage boundary. PlanStore can follow once you introduce the operations layer and want to inject a plan store there.

---

## 5. Agent runtime as an interface (low priority)

**Current:** `agent.Runtime` is a struct; execution and CLI/TUI call `agent.NewRuntimeFromEnv()` and pass it into `LaunchAgent` / `RunExecute`. OVERVIEW.md describes a “pluggable runtime.”

**Recommendation:** If you expect multiple backends (e.g. another CLI, or an HTTP API), define a small interface, e.g.:

```go
type Runner interface {
    Run(ctx context.Context, pack ContextPack) (RunRecord, error)
}
```

Current launcher + runtime becomes the default implementation. Execution and operations take `Runner` instead of `agent.Runtime`. No need to do this until you have a second implementation.

---

## 6. Execution package structure (documentation / light refactor)

**Current:** `internal/execution` contains context, execute, launcher, lifecycle, query, questions, resume, runner, selector, storage, types. All are related but mix “run execution,” “run persistence,” “plan lifecycle,” and “resume flow.”

**Recommendation:** No need to split packages immediately. Add a short sub-section in the package README (or a single doc) that groups files:

- **Run execution:** `context`, `execute`, `launcher`, `runner` – build context, launch agent, run loop.
- **Run persistence:** `storage`, `query` – save/list/load run records.
- **Plan lifecycle:** `lifecycle` – update task status and persist plan.
- **Resume:** `resume`, `questions` – answers, continuation context.

Optionally move `atomicWriteFile` / run-specific write logic into a single place (e.g. `storage.go`) if it’s duplicated with `plan/atomic.go`. This keeps the package cohesive while making the mental model explicit.

---

## Summary

| Recommendation                  | Impact  | Effort  | When to consider                   |
| ------------------------------- | ------- | ------- | ---------------------------------- |
| Explicit config / working dir   | Medium  | Low     | Soon; helps tests and future paths |
| Single operations layer         | High    | Medium  | When fixing parity or adding flows |
| Split TUI model                 | Medium  | Medium  | When model.go becomes painful      |
| Run/plan store interfaces       | Low now | Low–Med | With operations layer or for tests |
| Agent Runner interface          | Low     | Low     | When adding a second agent backend |
| Execution package documentation | Low     | Low     | Anytime                            |

Implementing (1) and (2) gives the largest benefit: one place that defines “where things live” and one place that defines “what execute/resume/generate/refine do,” with CLI and TUI as thin IO layers on top.
