# Execution pane visible during run + live stdout/stderr streaming

**status:** incomplete

## 1. Goals

- **View execution pane during run** — User can press `t` to switch to the Execution tab (Details ↔ Execution) even while an execute or resume action is in progress. Today `t` is ignored when `actionInProgress` is true.
- **Stream agent output** — Stdout and stderr from the agent (and, where relevant, from the execute/resume orchestration) are streamed into the Execution pane in real time so the user can watch progress without waiting for completion.
- **No change to completion behavior** — When the run finishes, the Execution pane continues to show the same data from the run record (and any live buffer can be discarded or merged). CLI behavior is unchanged.

Non-goals: streaming for plan generate/refine (only execute/resume); changing how the agent subprocess is launched; necessarily streaming set-status (can be later).

## 2. Current state

### 2.1 TUI key handling

- **`internal/tui/model.go`** — On `tea.KeyMsg`, `case "t":` toggles `tabMode` between `TabDetails` and `TabExecution` only when `m.actionMode == ActionModeNone` and `!m.actionInProgress`. Otherwise it returns without toggling. So during execute/resume the user cannot switch to the Execution tab.

### 2.2 Execution invocation

- **`internal/tui/action_wrappers.go`** — `ExecuteCmdWithContext` and `ResumeCmdWithContext` are `tea.Cmd`s that run synchronously: they call `execution.RunExecute` / `execution.RunResume` and return a single `ExecuteActionComplete` when done. No streaming; output is only summarized in the completion message.
- **`internal/execution`** — Run loop and `LaunchAgent` run to completion (or until context cancel). Agent stdout/stderr are consumed by the launcher (e.g. for run records or logging) but are not exposed incrementally to the TUI.

### 2.3 Execution pane rendering

- **`internal/tui/execution_view.go`** — `RenderExecutionView(model)` reads from `model.runData`. It shows "Active Run" (task, status, elapsed), "Log Output" (stdout/stderr from the active run record via `active.Stdout`, `active.Stderr`), and "Task Summary". Logs are only present in run records after a run completes or transitions (e.g. waiting_user). There is no live buffer for in-progress output.

## 3. Target state

### 3.1 Allow `t` during execute/resume

- When the user presses `t`, allow toggling `tabMode` even if `actionInProgress` is true. Optionally restrict this to execute/resume only (so plan generate/refine still block `t` if desired). No new message types; just relax the guard in `model.go` for `case "t"`.

### 3.2 Live output buffer in the model

- Add state for "live" stdout/stderr for the current run, e.g. `liveExecutionStdout`, `liveExecutionStderr` (or a single `liveExecutionOutput` with a simple stream-type tag). Cleared when the run completes or is cancelled; optional: merge final snapshot into the run record or discard (run record already gets final output from execution layer).
- Execution pane: when `actionInProgress` is true and the current action is execute or resume, render the live buffer (and optionally still show "Active Run" / task summary from run data or from in-memory progress). When not in progress, keep current behavior (render from `model.runData` only).

### 3.3 Streaming path from execution to TUI

- **Execution layer** — Provide a way to stream stdout/stderr to the TUI:
  - Option A: Add an optional `StreamWriter` (or `StdoutWriter` / `StderrWriter`) to execute/resume config; the runner pipes agent (and optionally orchestration) output to this writer. TUI supplies a writer that enqueues chunks and sends a message to the program.
  - Option B: Add an optional callback or channel (e.g. `OnOutput(chunk []byte, stream string)`) that the runner calls as output is read; TUI subscribes and maps chunks to a Tea message.
- **TUI** — Start execute/resume in a goroutine (as today with in-process execution). The goroutine calls the same runner API with a streaming hook. The hook sends chunks (e.g. via a channel). A `tea.Cmd` subscribes to that channel and, for each chunk, sends a message (e.g. `ExecutionOutputChunk{Stdout string, Stderr string}` or `StreamChunk{Text string, IsStderr bool}`) into the program. `Update` handles the message by appending to `liveExecutionStdout` / `liveExecutionStderr` and returns the same Cmd again (or a batch with the next subscription read) so that further chunks keep arriving until the run ends.
- **Bubble Tea** — Use a long-lived Cmd that reads from the channel and sends messages; when the channel is closed (run finished), the Cmd can send a final message or no-op and stop. Ensure only one streaming Cmd is active per run.

### 3.4 Agent output source

- Today the agent is a subprocess; the execution layer (e.g. `LaunchAgent` or the runner) reads its stdout/stderr. That read path must be changed to optionally copy each read to the streaming hook (or writer) in addition to (or instead of) buffering for the run record. So: execution package gains a configurable output sink; runner passes it through when launching the agent and when writing run records.

## 4. Design details

### 4.1 Message type

- New message type, e.g. `ExecutionOutputChunk struct { Stdout, Stderr string }` (or one message per stream). Model appends to `liveExecutionStdout` / `liveExecutionStderr`. View uses these when `actionInProgress` and action is execute/resume.

### 4.2 When to clear the buffer

- On `ExecuteActionComplete` (and on cancel before completion), set `liveExecutionStdout = ""`, `liveExecutionStderr = ""`. Optionally persist final output to run record in execution layer so the Execution pane then shows it from `runData` as it does today.

### 4.3 Execution view when in progress

- If `actionInProgress` and we have live buffer content, show "Active Run" (task ID can come from run data or from a "current task" progress event if added) and "Log Output" from the live buffer. Same layout as today; only the data source changes. Task Summary can still use `model.runData` and `execution.ReadyTasks(model.plan)`; if run data is updated mid-run (e.g. task started), that can be reflected.

### 4.4 Runner API extension

- Extend `ExecuteConfig` and `ResumeConfig` (or equivalent) with optional streaming:
  - e.g. `OnOutput func(stdout, stderr []byte)` called whenever output is read from the agent (or from the process that runs the agent), or
  - `OutputWriter io.Writer` (or two writers) that the runner copies agent stdout/stderr to.
- Runner and `LaunchAgent` (or equivalent) must pipe subprocess stdout/stderr through this hook. Implementation: when starting the agent command, set `cmd.Stdout` / `cmd.Stderr` to an `io.Writer` that both writes to the existing buffer (for run record) and to the streaming sink (e.g. `io.MultiWriter`).

## 5. Tasks (checklist)

- [ ] **TUI:** Relax `case "t":` in `model.go` so `t` toggles tab even when `actionInProgress` is true (optionally only for execute/resume).
- [ ] **Model:** Add `liveExecutionStdout`, `liveExecutionStderr` (or equivalent); clear on `ExecuteActionComplete` and on cancel.
- [ ] **Execution:** Add optional streaming to runner config (writer or callback); in runner/launcher, pipe agent stdout/stderr through it (e.g. MultiWriter) while retaining current behavior for run records.
- [ ] **TUI:** When starting execute/resume, pass a streaming sink that enqueues chunks and triggers a Tea message (e.g. channel + Cmd that reads channel and sends `ExecutionOutputChunk`).
- [ ] **TUI:** Handle `ExecutionOutputChunk` in `Update`: append to live buffer, return next Cmd if needed.
- [ ] **Execution view:** When `actionInProgress` and live buffer non-empty, render log section from live buffer; otherwise keep current behavior (run record only).
- [ ] **Tests:** Unit test for `t` during action (tab toggles); optional test for streaming (mock runner calls OnOutput, assert model buffer grows); execution view test for in-progress vs completed display.

## 6. Success criteria

- User can press `t` during an active execute or resume and see the Execution pane.
- Execution pane shows live stdout/stderr from the agent (and optionally orchestration) while the run is in progress.
- When the run completes, Execution pane shows the same information as today (from run record); no regression.
- CLI execute/resume behavior and output are unchanged.