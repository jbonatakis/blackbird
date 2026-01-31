Spec: Normalize timestamps when accepting full plan from agent
status: complete

TITLE
Normalize createdAt / updatedAt when accepting a full plan from the agent
OBJECTIVE
When the agent returns a full plan (not a patch), the application must not trust agent-supplied timestamps. It must normalize createdAt and updatedAt so that (1) every item has valid, consistent timestamps, and (2) validation updatedAt >= createdAt never fails due to placeholder or wrong values from the model.
BACKGROUND / PROBLEM
For full-plan responses, responseToPlan returns *resp.Plan unchanged (CLI: internal/cli/agent_helpers.go, TUI: internal/tui/action_wrappers.go).
LLMs often emit a single placeholder (e.g. 2026-01-31T12:00:00Z) for every item’s createdAt and updatedAt.
Later, set-status or edit sets only updatedAt to the real current time; createdAt is never updated.
That can yield updatedAt < createdAt, which fails validation: “must be >= createdAt”.
SCOPE
In scope
When converting an agent response to a plan, if the response contains a full plan (resp.Plan != nil), normalize timestamps before returning the plan.
Normalization must ensure for every work item: createdAt and updatedAt are set from a single authoritative now (or equivalent rule below), and updatedAt >= createdAt.
Apply in both CLI and TUI code paths that use full-plan agent responses.
Standardization (CLI and TUI)
We want CLI and TUI to do the same thing. For this spec: (1) both must normalize full-plan timestamps the same way; (2) prefer implementing the normalization in shared code (e.g. in internal/plan or internal/agent) so both CLI and TUI call the same helper, rather than duplicating logic in internal/cli/agent_helpers.go and internal/tui/action_wrappers.go. That keeps behavior identical and avoids future drift.
Out of scope
Changing how patches apply timestamps (patch path already uses now and preserves existing createdAt where appropriate).
Changing validation rules (keep updatedAt >= createdAt).
Changing manual add/edit (they already set timestamps correctly).
REQUIREMENTS
Normalization rule (full plan only)
When accepting a full plan from the agent:
Use a single now value (e.g. time.Now().UTC() or the now already passed into the conversion function where available).
For every work item in the plan, set:
createdAt = now
updatedAt = now
So the saved plan never contains agent-supplied createdAt/updatedAt for full-plan responses.
Where to apply
Prefer shared code so CLI and TUI stay identical: add a normalization helper (e.g. in internal/plan) that, given a plan and a now time, returns a copy with every item's createdAt and updatedAt set to now. Then:
CLI: responseToPlan in internal/cli/agent_helpers.go. When resp.Plan != nil, call the shared normalizer (with the same now used elsewhere in that flow) and return the result instead of *resp.Plan.
TUI: responseToPlan in internal/tui/action_wrappers.go — when resp.Plan != nil, call the same shared normalizer (e.g. with time.Now().UTC()) and return the result.
If shared code is not introduced, both places must still implement the same rule (single now, set createdAt and updatedAt to now for every item).
Patch path unchanged
When the response contains a patch (resp.Patch), keep current behavior: merge into base plan and apply patch with existing ApplyPatch(..., now). No change to how patch ops set createdAt/updatedAt.
Idempotence / determinism
Normalization should be deterministic: same now for all items in that conversion. No per-item “current time” during the loop, so all items in that plan share the same createdAt and updatedAt.
ACCEPTANCE CRITERIA
After blackbird plan generate (or TUI equivalent) that returns a full plan, every item in the saved plan has createdAt == updatedAt and both equal the normalization time used in that run.
Subsequently running blackbird set-status <id> <status> (or any mutation that only updates updatedAt) never causes validation to fail with “must be >= createdAt”.
Plan validation (plan.Validate) still passes after normalization.
Patch-based flows (e.g. refine) continue to behave as today; only full-plan acceptance is changed.
NON-GOALS
Changing the schema or validation rules for createdAt/updatedAt.
“Repairing” existing plan files on disk (no one-time migration); this spec only affects new full-plan responses from the agent.
Prompt engineering to improve agent timestamp output (optional later improvement; this spec fixes the behavior in code).
DELIVERABLES
Normalization logic applied when resp.Plan != nil in both CLI and TUI responseToPlan (prefer shared helper so both call the same code).
Tests: (1) full-plan response produces a plan where every item has createdAt == updatedAt and validation passes; (2) a subsequent status update still passes validation.
DONE CRITERIA
Full-plan agent responses always produce plans with normalized timestamps.
No validation failure of the form “updatedAt must be >= createdAt” when the only user action after a generated plan was updating status or editing a task.