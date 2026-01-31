status: complete

Implementation update (2026-01-31)
- Execute/resume orchestration now lives in the shared runner (`internal/execution.RunExecute` / `RunResume`).
- The TUI runs execute/resume in-process with a cancellable context; quit/ctrl+c invokes the cancel func to stop the runner and agent subprocess.
- Execute/resume UI actions now send completion messages directly instead of shelling out to `blackbird execute` or `blackbird resume`.

1. Goals
Single orchestration process: The process that runs the TUI (or CLI) runs the execute/resume loop in-process. Only the agent runs as a subprocess.
No orphaned runs: Quitting the TUI or hitting Ctrl-C cancels a shared context; the execute loop and agent subprocess stop. No “blackbird execute” left running.
One code path: Execute and resume logic live in a shared, callable API used by both CLI and TUI. No “TUI spawns CLI binary” for execute/resume.
Consistent with plan generate: TUI already runs plan generate in-process (GeneratePlanInMemory). Execute and resume should follow the same pattern.
Non-goals for this spec: changing how the agent is invoked (it remains a subprocess via execution.LaunchAgent), or necessarily moving set-status / plan refine in-process (can be a later phase).
2. Current State
2.1 Process Chain
CLI blackbird execute: One process. Uses signal.NotifyContext; runs execute loop; calls execution.LaunchAgent(ctx, ...); agent is subprocess. Context cancellation kills the agent. No orphan.
TUI Execute: Two processes. TUI runs exec.Command(exe, "execute") and waits. Child runs the same execute loop and spawns the agent. So: TUI → blackbird execute → agent. If the TUI exits first, the execute process can be orphaned.
Resume: Same pattern. TUI runs blackbird resume <taskID>; that process runs resume flow and launches the agent.
2.2 Where Logic Lives
Execute loop: internal/cli/execute.go — load plan, loop on ready tasks, build context, update status, execution.LaunchAgent, save run, update status by result. Uses planPath(), execution.ReadyTasks, execution.BuildContext, execution.LaunchAgent, execution.SaveRun, plan mutations.
Resume flow: internal/cli/resume.go — load plan, build context for task, execution.LaunchAgent, save run, update status. Same primitives.
Shared packages: internal/plan, internal/execution, internal/agent. CLI is the only caller of the execute/resume orchestration; TUI does not import internal/cli.
2.3 TUI Invocation Today
internal/tui/action_wrappers.go: ExecuteCmd() / ResumeCmd() / SetStatusCmd() call runCommand(args) → exec.Command(exe, args...).Run(). No handle to the child; no way to cancel it from the TUI.
3. Target State
3.1 Process Model
CLI blackbird execute: Unchanged at the process level. One process; execute loop runs in that process; only the agent is a subprocess. Implementation changes: CLI calls a shared runner API instead of owning the loop in cli/execute.go.
TUI Execute / Resume: One process (the TUI). Execute or resume runs in a goroutine via the same runner API. Context is tied to TUI lifecycle (e.g. cancelled when the program exits or user quits). No blackbird execute or blackbird resume subprocess. Agent remains the only subprocess.
Result: Process → (execute/resume loop in-process) → agent subprocess. No subprocess calling a subprocess for orchestration.
3.2 Shared Runner API
Location: internal/execution (or a dedicated internal/run if you prefer to keep execution focused on launch/context/storage).
Responsibilities:
Run the execute loop (or a single resume step) with a given context.Context.
Use existing building blocks: execution.BuildContext, execution.LaunchAgent, execution.SaveRun, plan load/update (from plan + possibly helpers in CLI or execution).
Support optional callbacks or channels for progress (e.g. “task X started”, “task X finished”, “run record”) so the TUI can update the UI without parsing stdout.
Return when the loop ends, context is cancelled, or a terminal state (e.g. waiting for user, no ready tasks, error).
CLI and TUI both call this API; neither implements the loop itself.
4. API Design
4.1 Execute
Option A – Runner in internal/execution
Add something like RunExecute(ctx, cfg ExecuteConfig) (Result, error).
ExecuteConfig: Plan path (or preloaded graph + path for persistence), runtime (agent config), and optional hooks.
Hooks (optional): e.g. OnTaskStart(taskID string), OnTaskEnd(taskID string, record RunRecord), or a single OnProgress(Event) channel. Default: no-op.
Result: Final state: last task (if any), whether stopped due to context cancellation, “waiting for user,” “no ready tasks,” or error. Enough for CLI to print and TUI to show a message or refresh.
Option B – Keep orchestration in CLI, but callable
Add cli.RunExecute(ctx, planPath string, stdout io.Writer) error (or with a progress callback). TUI would import internal/cli and call this from a goroutine. Less ideal: TUI depends on CLI and stdout/callback design can get awkward.
Recommendation: Option A. Keep orchestration in internal/execution (or internal/run) so both CLI and TUI depend only on plan + execution (+ agent), and the “execute loop” is a first-class concept in the execution layer.
4.2 Resume
Same idea: e.g. RunResume(ctx, cfg ResumeConfig) (RunRecord, error) in the same package.
ResumeConfig: Plan path, task ID, runtime, optional progress hook.
Single task run; no loop. Used by CLI and TUI.
4.3 Context Contract
All runner functions take context.Context as the first argument.
When ctx is cancelled (TUI quit, Ctrl-C, or explicit “Stop” later), the runner must:
Stop the loop (or not start the next task) and return promptly.
Rely on execution.LaunchAgent using CommandContext(ctx, ...) so the agent process is killed when the context is cancelled.
No long-running work after ctx.Done() except cleanup (e.g. saving a “cancelled” run record if desired).
4.4 Plan and Status Mutations
Today the execute/resume logic in CLI calls plan IO and status updates (e.g. execution.UpdateTaskStatus(path, taskID, status) or equivalent). The runner API should accept a plan path and perform the same updates so behavior stays consistent.
If plan loading is currently CLI-specific (planPath() etc.), either:
Move “resolve plan path / load plan” into a small shared helper (e.g. in plan or execution), or
Pass a preloaded graph + path into the runner and let the runner do status updates and saves. Spec should pick one and document it.
5. CLI Changes
5.1 Execute
In internal/cli/execute.go: Create ctx with signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) as today. Build ExecuteConfig (plan path from planPath(), runtime from agent.NewRuntimeFromEnv(), optional stdout progress hook that prints “starting X” / “completed X”).
Call execution.RunExecute(ctx, cfg) (or the chosen name). On return, map Result to current CLI behavior: print “execution interrupted” on cancel, “no ready tasks remaining,” or per-task messages. No change to user-visible behavior; only the implementation uses the shared runner.
5.2 Resume
In internal/cli/resume.go: Same pattern. ctx from signal.NotifyContext; build ResumeConfig; call execution.RunResume(ctx, cfg); then print success/failure/waiting based on the returned record. No subprocess.
6. TUI Changes
6.1 Context and Lifecycle
Execute: When the user triggers Execute (e.g. “e” from home), the TUI should:
Create a cancellable context (and its cancel func) for “this run.” Store the cancel func (e.g. on the model or in a package-level registry) so it can be invoked on quit or on a future “Stop” action.
Start a goroutine that calls execution.RunExecute(ctx, cfg). Pass a progress callback or channel so the TUI can update actionOutput or refresh run data (e.g. “Task X started”, “Task X completed”).
When the goroutine returns, send a message (e.g. ExecuteActionComplete) with success/failure/cancelled and any summary. Set actionInProgress = false and clear the stored cancel func.
Resume: Same idea: context for the run, goroutine calling execution.RunResume(ctx, cfg), message on completion.
6.2 Quit / Ctrl-C
When the TUI is about to quit (e.g. tea.Quit returned after Ctrl-C or “q”):
Before exiting, call the stored cancel func if a run is in progress. That cancels the context passed to RunExecute/RunResume, so the runner returns and the agent subprocess is killed by CommandContext.
No need to track or kill a “blackbird execute” subprocess; there is none.
6.3 Remove Subprocess Invocation for Execute and Resume
Remove ExecuteCmd() / ResumeCmd() that use runCommand([]string{"execute"}) or runCommand([]string{"resume", taskID}).
Replace with TUI-specific commands that:
Build ExecuteConfig/ResumeConfig (plan path from current dir or model, runtime from agent.NewRuntimeFromEnv()),
Create context + cancel,
Start the runner in a goroutine and send a completion message when it returns.
Optionally: use a small helper in internal/tui that wraps “run with context + cancel on quit” so key handling is in one place.
6.4 set-status and plan refine (optional in this spec)
Can remain as subprocess for now (runCommand([]string{"set-status", ...}) etc.). Bringing them in-process can be a follow-up: either call into CLI helpers or move their logic into a shared package and call from TUI.
7. Execution Package (or Run Package) Layout
7.1 New or Extended Types
ExecuteConfig: Plan path, agent runtime, optional progress callback/channel.
ResumeConfig: Plan path, task ID, agent runtime, optional progress callback.
ExecuteResult (or RunExecuteResult): e.g. StoppedReason (completed / cancelled / no ready tasks / waiting user / error), last task ID if any, error if failed.
Progress event type (optional): e.g. TaskStarted, TaskEnded(record RunRecord) so the TUI can refresh run list or show live output.
7.2 New Functions
RunExecute(ctx context.Context, cfg ExecuteConfig) (ExecuteResult, error)
Load plan from cfg.PlanPath (or accept preloaded graph). Loop: check ctx; get ready tasks; if none, return “no ready tasks”; take first task; build context; update status to in-progress; call LaunchAgent(ctx, ...); save run; update status from record; if waiting/failed/done, break or return; if ctx cancelled, return “cancelled.” Use cfg progress hook if provided.
RunResume(ctx context.Context, cfg ResumeConfig) (RunRecord, error)
Load plan; build context for cfg.TaskID; call LaunchAgent(ctx, ...); save run; update task status from record; return record and error.
7.3 Plan Path and Loading
Document where plan path comes from: CLI uses planPath() (cwd + default filename). TUI should use the same convention (e.g. cwd when Execute was pressed) or a path stored on the model. Runner should not depend on internal/cli; so either pass path + use plan load, or pass preloaded graph + path for writes.
8. Edge Cases and Behavior
Context cancelled mid-task: Runner should return as soon as practicable; LaunchAgent’s CommandContext will kill the agent. Optionally mark the run record as cancelled or leave as “running” and rely on “no active process.” Spec should state the chosen behavior.
TUI quit while execute is running: Cancel func is called; context cancelled; runner returns; agent killed. No orphan.
Double Execute: If the user triggers Execute twice, either ignore the second while actionInProgress is true, or cancel the first and start the second. Spec should pick one (recommend: ignore or “already running” message).
Resume with invalid task ID: Same as today: runner or plan layer returns error; TUI shows error in action output.
Plan file changed on disk during run: Either document “plan is read at start of run” or add a note for a future “reload plan” behavior. No need to support mid-run reload in this spec.
9. Testing
Unit tests for runner: Test RunExecute / RunResume with a cancelled context (runner returns quickly; no infinite loop). Test with mock or real plan + fake agent (e.g. short-lived subprocess that exits 0) to assert one task run and status updates. Test “no ready tasks” and “waiting for user” outcomes if feasible.
CLI tests: Existing execute/resume CLI tests should pass; only the implementation (call into runner) changes. Add or adjust tests that assert SIGINT cancels the run (context cancelled, runner returns).
TUI tests: Test that triggering Execute sets actionInProgress and that a completion message clears it. Optionally test that quit while “in progress” invokes cancel (e.g. mock or integration test). No need to test full agent run in TUI tests.
10. Migration Order
Add runner API in internal/execution (or internal/run): Implement RunExecute and RunResume by moving logic from cli/execute.go and cli/resume.go, taking context and config. Add tests.
Switch CLI to runner: Refactor runExecute and runResume to call the new API. Keep CLI tests green.
Switch TUI to runner: Replace ExecuteCmd/ResumeCmd subprocess calls with in-process runner + context + cancel-on-quit. Add TUI lifecycle handling (store cancel, call on quit).
Remove dead code: Delete any subprocess-only paths for execute/resume in TUI; tidy runCommand usage if it’s now only used for set-status/plan refine.
Document: Update any docs or comments that describe “TUI runs blackbird execute” to “TUI runs execute in-process.”
11. Success Criteria
No exec.Command(exe, "execute") or exec.Command(exe, "resume", ...) from the TUI.
Quitting the TUI (or Ctrl-C) while execute or resume is running does not leave a lingering “blackbird execute” or “blackbird resume” process.
CLI behavior (including Ctrl-C and “execution interrupted”) unchanged.
Execute and resume logic live in one place (runner API); CLI and TUI both call it.
Existing tests pass; new tests cover context cancellation and runner outcomes.
12. Out of Scope (for this spec)
Changing how the agent is started (still execution.LaunchAgent with CommandContext).
Moving set-status or plan refine in-process (can be a later spec).
Adding a “Stop” button in the TUI (optional follow-up; cancel func is already there).
Streaming agent stdout/stderr into the TUI in real time (optional; progress hooks can be extended later).
