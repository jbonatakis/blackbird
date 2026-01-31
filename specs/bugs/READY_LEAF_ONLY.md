Spec: Only leaf tasks eligible for execution
status: complete

TITLE
Only leaf tasks eligible for execution (exclude parent/container tasks)
OBJECTIVE
Ensure that parent-level tasks—items that have children and act as containers for smaller work—are never dispatched to the agent. Execution (CLI and TUI) must operate only on leaf tasks (tasks with no children), so work is tackled one concrete piece at a time and "do the whole thing" is never invoked for a container.
BACKGROUND / PROBLEM
Plans use a tree: parent tasks have non-empty childIds and encompass the entirety of a feature; their children are the actual work items (run_metadata, cli_commands, tui_actions, tests_docs).
ReadyTasks in internal/execution/selector.go considers a task ready when it is todo and has all deps satisfied. It does not consider parent/child structure. So a parent task with status todo and no deps can be returned as "ready" and dispatched.
Dispatching a parent is equivalent to "do the whole thing," which goes against the application goal of breaking work into chunks and tackling one piece at a time.
The CLI pick command already defaults to leaf tasks only (leafIDs in internal/cli/cli.go); execution does not, so behavior is inconsistent.
SCOPE
In scope
Changing the definition of "ready for execution" so that only leaf tasks (items with len(childIds) == 0) are eligible. Apply this in ReadyTasks so that CLI execute, TUI execute, and any UI that uses "ready count" all treat parent/container tasks as not executable.
Out of scope
Changing the plan schema (parentId/childIds). Changing how pick works (it already uses leaves by default). Adding a new "container" type or field; we use "has children" as the criterion.
REQUIREMENTS
Ready = todo + deps satisfied + leaf
A task is eligible for execution only if: (1) status is todo, (2) all deps are satisfied (UnmetDeps empty), and (3) the task is a leaf (len(it.ChildIDs) == 0). Parent/container tasks must never appear in the ready list.
Where to apply
internal/execution/selector.go: In ReadyTasks, after the existing status and UnmetDeps checks, add a check that skips any item where len(it.ChildIDs) > 0. No other call sites need to change; CLI execute and TUI both use ReadyTasks for "ready" count and task selection.
Consistency
"Ready" count in the home view, bottom bar, and execution view shall reflect only tasks that are actually dispatchable (i.e. ready leaves). No separate "ready but not executable" count is required.
NON-GOALS
Changing validation or schema for WorkItem. Deriving or auto-updating parent task status when children complete (future enhancement). Changing pick behavior (already leaf-scoped by default).
ACCEPTANCE CRITERIA
ReadyTasks returns no task that has non-empty childIds. Parent tasks never appear in the ready list regardless of status and deps.
blackbird execute and TUI "execute" only ever start a leaf task; no parent/container task is ever dispatched to the agent.
Existing tests for ReadyTasks are updated or extended so that a task with childIds non-empty is not considered ready even when todo and deps satisfied. New test: ready list includes only leaves.
Pick and list behavior unchanged for leaf-only display where already in use.
DELIVERABLES
Change in internal/execution/selector.go: exclude non-leaf tasks from ReadyTasks (skip when len(it.ChildIDs) > 0). Comment updated to state that only leaf tasks are eligible for execution.
Tests: (1) ReadyTasks excludes items with non-empty childIds; (2) ReadyTasks still returns leaf tasks that are todo and deps satisfied; (3) optional: integration or CLI test that execute never selects a parent task when a plan has both parents and leaves.
DONE CRITERIA
Parent/container tasks are never operated on by the program: they never appear in the ready list and are never dispatched. Only leaf tasks are eligible for execution in both CLI and TUI.