# Blackbird Pause/Resume + Restart Spec
status: incomplete

## Purpose

Add a **pause and resume** capability to Blackbird plan execution so a user can stop execution part way through and later continue **without rehydrating agent context**. Blackbird must leverage **agent-native resume features** for both Codex and Claude Code.

Additionally, provide an explicit **restart** capability for cases where native resume is unavailable or undesired (e.g., the provider no longer has the session).

This feature must work in both **CLI** and **TUI** and must **reuse the same core code paths** so behavior matches.

Repository: `https://github.com/jbonatakis/blackbird`

## Core Requirements

### 1) Pause via Ctrl+C

* **CLI**

  * While executing a plan, pressing **Ctrl+C** pauses the currently running task/run and then **exits the process**.
* **TUI**

  * While executing, pressing **Ctrl+C** pauses the currently running task/run and **remains in the TUI**.
  * After pausing, the TUI must display a banner/message: **“Paused. Ctrl+C again to quit.”**
  * Pressing **Ctrl+C again** quits the TUI (no additional state changes beyond quitting).

### 2) Pause scope

* Pausing stops **only the currently active task’s agent run**.
* After a pause occurs, Blackbird must **not automatically start any other tasks**, even if they are ready.
* Resuming or continuing execution requires explicit user action.

### 3) Resume semantics

* `blackbird resume <taskId>` must:

  * If the latest run for `taskId` is **paused**, resume that run using **agent-native resume**.
  * Else if the latest run is **waiting**, continue the existing waiting-state behavior (current semantics).
  * Else error (nothing to resume).

### 4) Restart semantics (backdoor)

Provide a new explicit operation: **restart a task from scratch**, creating a new run and starting a new agent session.

Motivation: If a paused run cannot be resumed (missing/invalid provider session ref, provider lost session history), users need a deterministic recovery path without manual file edits.

* CLI: add `blackbird restart <taskId>`
* TUI: add an explicit restart action (key/button) for selected task
* Restart must **never** happen automatically; it is a user choice.
* Restart must **preserve history**: old paused run remains in run history but is marked as superseded/linked to the new run.

### 5) Provider constraints

* Resume must use the **same provider** that started the paused run.
* **Codex**: must use native resume support (`codex exec resume <session>` or equivalent native mechanism).
* **Claude Code**:

  * Must **never** use `--continue`.
  * Must **always** resume from an explicit session identifier using `--resume <id|name>` or equivalent.
* If native resume cannot be performed (no session ref, invalid ref, provider error):

  * **For now**: treat as an error; require manual intervention or recommend `restart`.

### 6) Storage / environment constraints

* Pause/resume state must be stored in the existing run history storage (under `.blackbird/…` run records).
* Resuming requires the **same repo/workdir** as the original run. If current workdir does not match the recorded repo root, error.

### 7) Concurrency

* Only one active agent session at a time. Pausing/resuming applies to a single active run.

## Non-Goals

* Automatic continuation of execution after pause without explicit user command.
* Cross-repo resume.
* Multi-run concurrent execution.
* “Best effort” fallback resume modes (`claude --continue`, “resume last session” without explicit session ref).

## Definitions

* **Task**: a unit of work in a plan.
* **Run**: an execution attempt for a task; runs are persisted and visible via existing run history (`blackbird runs <taskId>`).
* **Paused run**: a run interrupted by user intent (Ctrl+C) with sufficient metadata to resume the provider session.
* **Resumable**: a paused run that has a provider session reference that Blackbird can pass to the provider’s native resume mechanism.
* **Restart**: start a new run from scratch (new provider session), leaving the paused run intact in history.

## User Experience

### CLI flows

#### Pause during execute

* User runs: `blackbird execute`
* While an agent is running, user presses Ctrl+C
* Behavior:

  * Current run transitions to `paused`
  * Run record is persisted with resumable metadata (if available)
  * Process exits
  * CLI prints instructions: `Paused. Resume with: blackbird resume <taskId>` and `Restart with: blackbird restart <taskId>` (restart message may be conditional on state)

#### Resume

* User runs: `blackbird resume <taskId>`
* If latest run is paused and resumable:

  * Blackbird resumes the provider session and continues the run to completion (or next state)
* If latest run is paused but not resumable:

  * Error explaining why and instructing user to use restart
* If latest run is waiting:

  * Keep existing waiting behavior

#### Restart

* User runs: `blackbird restart <taskId>`
* Blackbird creates a new run record and executes the task from its canonical prompt/spec.

### TUI flows

* While executing:

  * Ctrl+C pauses the active run and shows “Paused. Ctrl+C again to quit.”
* TUI provides:

  * **Resume** action (existing `u`) that now applies to paused and waiting tasks
  * **Restart** action (new) for paused/failed tasks (and optionally any task, gated by confirmation)
* TUI and CLI must route pause/resume/restart through the same core execution controller.

## Persistence Model

### Run record requirements

Each run record must contain enough information to:

* Identify its task
* Represent state transitions (running/paused/etc.)
* Record provider and provider session reference for native resume
* Enforce repo/workdir constraint
* Link restarts/resumes to predecessor runs

Minimum required fields (conceptual):

* `run_id` (string)
* `task_id` (string)
* `state` (enum: running, waiting, paused, succeeded, failed, canceled)
* `provider` (enum: codex, claude)
* `provider_session_ref` (string, optional but required for resumable paused runs)
* `resumable` (bool)
* timestamps: `created_at`, `updated_at`
* pause metadata:

  * `paused_at` (timestamp)
  * `pause_reason` (enum: user_interrupt)
* linkage:

  * `resumed_from_run_id` (optional)
  * `restart_of_run_id` (optional)
  * `superseded_by_run_id` (optional)
* environment guardrails:

  * `repo_root` (absolute path string)

### State transitions

* running → paused (on Ctrl+C)
* paused → running (on resume)
* paused → superseded (logical via `superseded_by_run_id` when restarted; state may remain paused but linked)
* running → waiting (existing behavior)
* running → succeeded/failed (existing behavior)
* waiting → running (existing behavior on resume/continue)

## Agent Integration Requirements

### Session handle capture

When starting an agent run, Blackbird must capture a stable **provider session reference** that can be used later to resume:

* **Codex**: capture the session identifier required by `codex exec resume <SESSION_ID>` (preferred).
* **Claude Code**: capture an explicit identifier that can be used with `claude --resume <SESSION>` (required). Never rely on “most recent” semantics.

If Blackbird cannot capture a stable session reference for a run:

* If the run is interrupted, persist it as `paused` with `resumable=false` and `provider_session_ref` empty, and `resume` must error.

### Resume execution

When resuming a paused run, Blackbird must:

* Load the paused run record
* Validate:

  * same provider
  * `resumable=true`
  * `provider_session_ref` present
  * current repo root matches `repo_root`
* Invoke provider-native resume with the stored ref and continue the task’s execution flow.

### Restart execution

Restart must ignore any prior provider session refs and start a new provider session, producing a new run record.

## Shared Core Architecture (CLI + TUI)

### Single execution controller

Introduce/ensure a single internal controller that owns execution lifecycle and is called by both CLI and TUI, with methods equivalent to:

* ExecuteReady(plan/task selection)
* PauseActiveRun(reason=user_interrupt)
* ResumeTask(taskId)
* RestartTask(taskId)

Constraints:

* Both CLI signal handling and TUI key handling must funnel into the same `PauseActiveRun` logic.
* Persistence must be performed by the controller (or beneath it) so UI layers cannot diverge.

### Process control

When pausing, Blackbird must attempt a graceful stop of the underlying agent subprocess:

* Send SIGINT/interrupt to the subprocess
* Allow a short grace period to flush/capture session ref if needed
* If the user hits Ctrl+C again (CLI or TUI) or if the agent doesn’t stop in time:

  * Force-kill the subprocess
  * Persist a paused run record as best effort (may be unresumable)

## CLI/TUI Command Surface

### CLI

* Existing:

  * `blackbird execute`
  * `blackbird resume <taskId>` (extend semantics per above)
  * `blackbird runs <taskId>` (must display paused state and resumable metadata)
* New:

  * `blackbird restart <taskId>`

### TUI

* Existing `u` resume behavior must be extended:

  * If selected task’s latest run is paused → resume
  * Else if waiting → current resume flow
* Add a Restart action (binding up to implementation; must be explicit and confirmed)

## Errors and Diagnostics

When resume fails (missing/invalid session ref, provider rejects resume, wrong repo root):

* Return a clear error indicating:

  * taskId/runId
  * provider
  * stored session ref (if any)
  * why resume could not proceed
  * recommended recovery: `blackbird restart <taskId>`

No automatic fallback to restart or “continue most recent session.”

## Implementation Notes / API Sketch (Go)

The spec does not mandate exact code structure, but the run record and controller API must be shared.

Illustrative Go types:

```go
type Provider string

const (
	ProviderCodex  Provider = "codex"
	ProviderClaude Provider = "claude"
)

type RunState string

const (
	RunRunning   RunState = "running"
	RunWaiting   RunState = "waiting"
	RunPaused    RunState = "paused"
	RunSucceeded RunState = "succeeded"
	RunFailed    RunState = "failed"
	RunCanceled  RunState = "canceled"
)

type PauseReason string

const (
	PauseUserInterrupt PauseReason = "user_interrupt"
)

type RunRecord struct {
	RunID  string `json:"run_id"`
	TaskID string `json:"task_id"`

	State    RunState `json:"state"`
	Provider Provider `json:"provider"`

	ProviderSessionRef string `json:"provider_session_ref,omitempty"`
	Resumable          bool   `json:"resumable"`

	RepoRoot string `json:"repo_root"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	PausedAt    string      `json:"paused_at,omitempty"`
	PauseReason PauseReason `json:"pause_reason,omitempty"`

	ResumedFromRunID string `json:"resumed_from_run_id,omitempty"`

	RestartOfRunID     string `json:"restart_of_run_id,omitempty"`
	SupersededByRunID  string `json:"superseded_by_run_id,omitempty"`
}
```

Agent provider interface concept (illustrative):

```go
type AgentSession interface {
	Provider() Provider
	SessionRef() string // stable ref usable for native resume; empty if unavailable
}

type AgentRunner interface {
	Start(ctx context.Context, prompt string) (AgentSession, error)
	Resume(ctx context.Context, sessionRef string, followup string) error
	Interrupt() error // best-effort SIGINT
	Kill() error      // force kill
}
```

## Acceptance Criteria

* Ctrl+C pauses execution per CLI/TUI requirements and persists a paused run record.
* After pausing, no further tasks are started automatically.
* `blackbird resume <taskId>` resumes paused runs using native provider resume features, and resumes waiting runs per existing behavior.
* Claude resume never uses `--continue`; only explicit session resume.
* Resume requires same repo root as recorded.
* If resume is not possible, it errors with actionable diagnostics and points to restart.
* `blackbird restart <taskId>` starts a new run from scratch and preserves history of the paused run (linked/superseded).
* CLI and TUI share the same pause/resume/restart execution code paths (controller-level), preventing divergent behavior.
