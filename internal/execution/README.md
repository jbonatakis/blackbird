# Execution Architecture

This package implements Phase 2 execution primitives:

- **Context building** (`BuildContext`): assembles task details, dependency summaries, and a
  project snapshot (prefers `.blackbird/snapshot.md`, then `OVERVIEW.md`, then `README.md`).
- **Run records** (`RunRecord` + `SaveRun`/`ListRuns`/`LoadRun`/`GetLatestRun`): persistent JSON
  artifacts stored under `.blackbird/runs/<taskID>/<runID>.json`.
- **Launcher** (`LaunchAgent`): executes the configured agent command with a context pack on stdin,
  captures stdout/stderr, detects AskUserQuestion output, and returns a `RunRecord`.
- **Lifecycle** (`UpdateTaskStatus`): validates status transitions and writes the plan atomically.
- **Resume** (`ResumeWithAnswer`): validates user answers and builds a continuation context.

## Flow Overview

1. `blackbird execute` loads the plan and selects ready tasks via `ReadyTasks`.
2. Each task is marked `in_progress`, context is built, and the agent is launched.
3. `RunRecord` is written to disk and plan status is updated to `done`, `failed`, or `waiting_user`.
   If execution review checkpoints are enabled, the run record is marked with a pending decision
   and execution halts until a decision is resolved.
4. `blackbird resume` prompts for answers and continues execution for waiting tasks.

## Execution Output

Agents are expected to modify the working tree directly (native CLI behavior).
Blackbird records stdout/stderr and exit codes for auditing.

Execution context includes a system prompt that authorizes non-destructive
commands and file edits without confirmation.

## Auto-Approve Flags

When launching headless runs, Blackbird appends provider-specific auto-approve
flags (unless `BLACKBIRD_AGENT_CMD` is set, which bypasses this logic):

- Codex: `exec --full-auto`
- Claude: `--permission-mode bypassPermissions`

## Streaming Output

If `BLACKBIRD_AGENT_STREAM=1` is set, stdout/stderr are streamed live while still being captured
in the run record.
