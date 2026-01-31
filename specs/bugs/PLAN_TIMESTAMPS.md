Spec: Normalize timestamps when accepting full plan from agent
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
CLI: responseToPlan in internal/cli/agent_helpers.go. When resp.Plan != nil, instead of return *resp.Plan, nil, build a plan from the agent’s structure but with normalized timestamps (e.g. clone and walk Items, setting both fields to now), then return that plan.
TUI: responseToPlan in internal/tui/action_wrappers.go. Same behavior: when resp.Plan != nil, normalize timestamps (using time.Now().UTC() or equivalent) before returning.
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
Normalization logic applied when resp.Plan != nil in both CLI and TUI responseToPlan.
Tests: (1) full-plan response produces a plan where every item has createdAt == updatedAt and validation passes; (2) a subsequent status update still passes validation.
DONE CRITERIA
Full-plan agent responses always produce plans with normalized timestamps.
No validation failure of the form “updatedAt must be >= createdAt” when the only user action after a generated plan was updating status or editing a task.