# Product Spec: Autonomous Task Execution Dispatch (Phase 2)
status: complete

## 1. Summary

Enable Blackbird to move from planning to execution by automatically selecting ready tasks from the plan and dispatching a headless coding agent (Claude Code, Codex, or other supported runtimes). The product becomes the execution control plane: it chooses what should run next, launches the agent with the correct context, tracks progress, and records durable run artifacts.

## 2. Problem & Why Now

Today, users must manually open an agent, feed it the plan, and ask it to implement tasks. This manual bridge is slow, error-prone, and undermines the core value of Blackbird: structured, dependency-aware execution. Automating task dispatch restores the intended workflow: plan once, then execute reliably with minimal human overhead.

## 3. Goals

- Make Blackbird the default way to execute plan tasks.
- Reduce user ceremony to a single command (start/stop execution).
- Ensure only safe, ready tasks are executed, with clear visibility and auditability.
- Support multiple agent providers without forcing users to change their plan.

## 4. Non-Goals

- Replacing task planning or editing workflows (Phase 1 already covers those).
- Real-time UI beyond the existing CLI output and run logs.
- Advanced multi-user coordination or distributed orchestration.
- Automatically merging or pushing code changes.

## 5. Core User Experience (What Users Do)

### 5.1 Start Execution

User runs a single command (e.g., “execute” or “run queue”). Blackbird:
- loads the plan,
- identifies ready tasks,
- selects the next task(s) to run,
- launches headless agent runs,
- streams or summarizes progress.

### 5.2 Observe and Intervene

Users can:
- see which task is running,
- view run status and logs,
- pause/stop execution,
- answer agent clarification questions when prompted.

### 5.3 Completion

When a task finishes, Blackbird:
- updates task status,
- records artifacts and run history,
- selects the next ready task until no runnable work remains.

## 6. Core Concepts

- **Execution Dispatcher**: the product behavior that selects which tasks to run and when.
- **Run Record**: a durable record of an agent execution (task, provider, timestamps, outcome, artifacts).
- **Ready Task Policy**: the rule set determining what can be executed automatically.
- **Human-in-the-Loop Gate**: clear pauses when agent needs clarification or confirmation.

## 7. Product Requirements

### 7.1 Task Selection and Dispatch

- Only tasks that are `ready` (deps satisfied) are eligible for automatic dispatch.
- The system must respect manual overrides (`blocked`, `skipped`).
- Default selection policy is simple and deterministic (e.g., smallest id order).
- Users can optionally constrain execution by tags or explicit task IDs.

### 7.2 Execution Control

- Users can start, pause, resume, and stop execution from the CLI.
- Execution can be configured to run one task at a time or multiple independent tasks in parallel (optional).
- If no tasks are ready, the system exits with a clear explanation.

### 7.3 Status Updates

- Each run must update task status through a defined lifecycle:
  - `queued` → `in_progress` → `done` or `failed` or `waiting_user`.
- Status transitions are durable and visible.

### 7.4 Human-in-the-Loop

- If the agent requests input, the run pauses in `waiting_user`.
- The user can answer directly via CLI and resume the run.
- Q/A must be recorded in the run history.

### 7.5 Context Integrity

- Each run uses a bounded, explicit context pack derived from:
  - task prompt,
  - project snapshot,
  - prior task artifacts (as applicable).
- The exact context pack used is recorded in the run record.

### 7.6 Artifacts and Auditability

- All runs produce:
  - outcome state,
  - timestamps and duration,
  - stdout/stderr logs,
  - summarized changes (if any),
  - agent provider metadata.
- Artifacts are stored in the repo for inspection and reproducibility.

### 7.7 Safety and Recovery

- A failed task does not auto-advance to dependents.
- Users can re-run a failed task after manual intervention.
- If execution is interrupted, Blackbird can resume without data loss.

### 7.8 Provider Flexibility

- Users select their provider once; Blackbird routes execution accordingly.
- Execution behavior should feel identical regardless of provider choice.

## 8. Definition of Done

The feature is complete when:

- A single CLI command can execute ready tasks end-to-end without manual agent invocation.
- Tasks are selected automatically based on readiness and status.
- Headless agent runs are launched and tracked with durable run records.
- Status updates and artifacts persist across restarts.
- Human-in-the-loop questions pause execution and resume correctly after user input.
- A user can view run history for any task, including logs and outputs.
- Execution stops safely when no runnable tasks remain or when user requests stop.

## 9. Success Metrics

- Time-to-first-task-run reduced to a single command.
- >90% of plan execution requires no manual agent launch.
- Users can resume execution after interruption without rework.

