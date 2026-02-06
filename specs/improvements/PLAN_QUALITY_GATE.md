# Plan Quality Gate (Deterministic Lint + Bounded Auto-Refine)
status: complete

## Purpose

Generated plans are often schema-valid but execution-weak: short descriptions, generic acceptance criteria, and prompts that require the execution agent to guess. This spec adds a quality gate after plan generation so plans are not only valid, but actionable.

The gate is a **hybrid** approach:

1. deterministic lint for stable quality checks
2. optional bounded auto-refine pass to improve failing plans

## Goals

1. Improve leaf-task execution readiness without requiring manual rewrite for every generated plan.
2. Keep behavior deterministic and testable; avoid pure “LLM judge” quality checks.
3. Reuse existing `plan_refine` flow for semantic improvements when lint fails.
4. Prevent infinite refinement loops with strict attempt limits.
5. Keep CLI and TUI behavior aligned.

## Non-Goals

1. Replacing schema validation in `internal/plan`.
2. Guaranteeing “perfect” plan quality in one pass.
3. Turning quality checks into rigid prose-length requirements.
4. Introducing unbounded autonomous re-generation loops.
5. Changing execution readiness semantics (leaf-only execution remains unchanged).

## High-Level Flow

For `plan generate` (CLI and TUI):

1. Agent returns proposed plan (current behavior).
2. Run deterministic plan quality lint on proposed plan.
3. If no blocking findings:
   - proceed with existing accept/save flow.
4. If blocking findings:
   - run bounded auto-refine requests (up to configured max) using lint findings as a structured change request.
   - re-run lint after each refined plan.
5. If blocking findings remain after max auto-refine passes:
   - do not silently accept.
   - show findings and let user explicitly choose next step (revise manually, accept anyway, or cancel).

Default bound:

- controlled by config key `planning.maxPlanAutoRefinePasses`
- default `1`
- `0` disables auto-refine passes

## Quality Lint Model

Lint output should be structured and deterministic:

```go
type PlanQualityFinding struct {
    Severity  string // "blocking" | "warning"
    Code      string // stable rule code
    TaskID    string
    Field     string // description | acceptanceCriteria | prompt | task
    Message   string
    Suggestion string
}
```

Blocking findings prevent automatic acceptance.
Warnings are surfaced but do not block save.

## Deterministic Rules

Rules primarily target **leaf tasks** (`len(childIds) == 0`), since these are execution units.

### Blocking rules

1. `leaf_description_missing_or_placeholder`
   - Description empty or placeholder-like (“todo”, “tbd”, “implement feature”, etc.).
2. `leaf_acceptance_criteria_missing`
   - Acceptance criteria empty.
3. `leaf_acceptance_criteria_non_verifiable`
   - Criteria are purely vague/non-testable (e.g., only “works well”, “is robust”).
4. `leaf_prompt_missing`
   - Prompt empty.
5. `leaf_prompt_not_actionable`
   - Prompt lacks concrete implementation direction and expected completion signal.

### Warning rules

1. `leaf_description_too_thin`
   - Description appears underspecified.
2. `leaf_acceptance_criteria_low_count`
   - Fewer than recommended criteria for non-trivial work.
3. `leaf_prompt_missing_verification_hint`
   - Prompt lacks validation/testing direction.
4. `vague_language_detected`
   - Terms like “improve/fix/handle” without explicit expected behavior.

### Important constraint

Word count may be used only as a weak signal. It must not be the only determinant for actionable vs non-actionable quality.

## Auto-Refine Request Construction

When blocking findings exist:

1. Build a deterministic change request from findings (grouped by task ID).
2. Include explicit instruction to preserve IDs, hierarchy, and dependencies unless lint requires structural adjustment.
3. Ask specifically for richer leaf descriptions, objective criteria, and execution-ready prompts.

Example refine intent (conceptual):

- “For task X, rewrite description to include intent/scope/constraints; add objective, testable acceptance criteria; rewrite prompt to include implementation target and completion verification.”

## Save/Accept Policy

1. Auto-save is allowed only when blocking findings are absent.
2. If blocking findings remain after bounded auto-refine:
   - CLI and TUI must show findings before save.
   - user can explicitly override and save anyway (with clear warning).

## Configuration and TUI Settings

Add a new config option:

- `planning.maxPlanAutoRefinePasses` (int)

Semantics:

1. Number of automatic `plan_refine` passes attempted after lint reports blocking findings.
2. `0` means no automatic refinement; findings are shown immediately to the user.
3. Value is clamped to bounds `0`..`3`.

Config behavior:

1. Same precedence model as existing config keys: project > global > default.
2. Included in resolved runtime config used by CLI and TUI plan generation flows.

TUI Settings integration:

1. Add a row in Settings for `planning.maxPlanAutoRefinePasses`.
2. Expose/edit in Local and Global columns like existing int settings.
3. Show applied source and out-of-range clamp warnings consistent with existing settings behavior.

## CLI UX

`blackbird plan generate`:

1. Print deterministic quality summary for both initial and final findings.
2. If auto-refine runs, print pass progress (`quality auto-refine pass X/Y`).
3. Print blocking and warning finding details for the final plan when present.
4. On remaining blocking findings, require explicit choice:
   - `revise` (manual refine prompt),
   - `accept_anyway`,
   - `cancel`.

## TUI UX

In the plan review modal flow:

1. Show quality summary panel with initial/final blocking+warning counts and key findings.
2. If auto-refine ran, indicate pass count and whether blocking findings remain.
3. If blocking findings remain, require explicit override via `Accept anyway`; default selection shifts away from accept (to `Revise`, or `Reject` when revision limit is reached).

## Shared Implementation Shape

Introduce shared package:

- `internal/planquality`

Core functions:

1. `Lint(g plan.WorkGraph) []PlanQualityFinding`
2. `HasBlocking(findings []PlanQualityFinding) bool`
3. `Summarize(findings []PlanQualityFinding) ...`
4. `BuildRefineRequest(findings []PlanQualityFinding) string`

Primary call sites:

1. CLI plan generate flow in `internal/cli/agent_flows.go`
2. TUI generate/review flow in `internal/tui/action_wrappers.go` and model review path
3. Config plumbing in `internal/config` (types, resolve, option registry, settings persistence/resolution)

## Testing Requirements

1. Unit tests for each lint rule (blocking + warning).
2. Unit tests for deterministic refine request construction.
3. CLI flow tests:
   - no findings -> normal accept path
   - blocking findings -> auto-refine pass -> success
   - blocking findings remain -> explicit user decision required
4. TUI flow tests for quality summary display and override path.
5. Determinism tests ensuring same plan yields same findings order/content.

## Rollout Notes

Initial rollout can enable lint + reporting first, then enforce blocking behavior once rule quality is stable. Auto-refine can be default-on after deterministic lint proves low false-positive rates.

## Done Criteria

1. `plan generate` runs deterministic lint before acceptance and reports initial/final quality summaries.
2. Blocking findings trigger bounded auto-refine controlled by `planning.maxPlanAutoRefinePasses` (default `1`, clamped `0`..`3`, `0` disables auto-refine).
3. Remaining blocking findings require explicit override to save in both interfaces (`accept_anyway` in CLI, `Accept anyway` in TUI).
4. Config loading/resolution and TUI Settings support `planning.maxPlanAutoRefinePasses` with local/global/default/applied behavior and clamp warnings.
5. CLI and TUI share quality-gate orchestration (`internal/plangen` + `internal/planquality`) and deterministic refine-request construction.
6. Tests cover deterministic lint/summary output, bounded loop behavior, config plumbing, and CLI/TUI decision paths for override and revision.
