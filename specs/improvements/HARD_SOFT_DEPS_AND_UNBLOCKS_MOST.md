status: pending
---
name: hard-soft-deps-unblocks-most
overview: Add soft dependencies (separate from hard deps), enforce that a task appears in at most one of hard or soft deps per dependent, and order ready tasks by "unblocks most" (prefer the task that the most other not-done tasks depend on, hard or soft).
todos:
  - id: schema-soft-deps
    content: Add SoftDeps to WorkItem; validation for mutual exclusivity and references
    status: pending
  - id: readiness-hard-only
    content: Readiness uses only hard deps; UnmetDeps / depsSatisfied for hard only; soft deps do not block
    status: pending
  - id: dependents-hard-soft
    content: Dependents (and reverse lookups) consider both hard and soft deps
    status: pending
  - id: unblocks-most-selector
    content: Ready task ordering by unblocks-most; runner/CLI/TUI use first of ordered list
    status: pending
  - id: cycle-dep-rationale
    content: Cycle detection remains hard-deps only; depRationale may reference hard or soft deps
    status: pending
  - id: tests-docs
    content: Tests for readiness, ordering, validation; update docs/OVERVIEW or README if needed
    status: pending
---

# Hard vs Soft Dependencies and "Unblocks Most" Task Ordering

## Goals

- **Two dependency lists**: Each task has **hard** deps (`deps`) and **soft** deps (`softDeps`). A given task ID may appear in **at most one** of these for a given dependent (mutually exclusive per dependent).
- **Readiness**: Only **hard** deps must be satisfied for a task to be ready. Soft deps do **not** block readiness; they express "ideally this happens first."
- **Ordering among ready tasks**: When multiple tasks are ready, prefer the one that **unblocks the most** other (not-done) tasks—i.e. the task that the most other tasks list as a hard or soft dep. Tie-break with a stable secondary sort (e.g. task ID).
- **Backward compatibility**: Existing plans have only `deps`; `softDeps` is optional (default empty). No task can be in both lists for the same dependent.

## Scope

**In scope**

- Schema: add `softDeps` to `WorkItem` (array of strings, same shape as `deps`).
- Validation: `softDeps` required (use `[]` if none); each ID in `deps` and `softDeps` must reference an existing item; no duplicate within `deps`; no duplicate within `softDeps`; **no ID may appear in both `deps` and `softDeps`** for the same item.
- Readiness: a task is ready iff it is a leaf, todo, and all **hard** deps are satisfied. Soft deps are ignored for readiness.
- Task selection order: among ready tasks, sort by **unblocks-most** (descending count of other not-done tasks that have this task as hard or soft dep), then by task ID for stability.
- Cycle detection: remains on **hard** deps only. Soft deps are not followed for cycle detection (soft cycles are allowed).
- `depRationale`: keys may refer to either `deps` or `softDeps` (must appear in at least one).

**Out of scope (this spec)**

- Priority or other ordering signals; unblocks-most + ID tie-break only.

## Definitions

- **Hard dep**: A prerequisite that **must** be done before the dependent task is considered ready. Stored in `deps`. Current behavior.
- **Soft dep**: A prerequisite that we **prefer** to have done before the dependent runs, but does **not** block readiness. Stored in `softDeps`. When choosing among ready tasks, we still prefer to run the soft-dep first when possible (via unblocks-most).
- **Unblocks-most**: For a ready task T, the number of **other** tasks (not T) that (a) list T in their `deps` or `softDeps`, and (b) are not yet done. Higher count = "more others are waiting on T" = prefer T first.

## Schema

**WorkItem** (add one field):

- `softDeps` — `[]string`, same semantics as `deps` but soft. Required in schema (use `[]` if none). JSON: `"softDeps": []` or `"softDeps": ["id1", "id2"]`.

**Invariants**

- For each item, `deps` and `softDeps` must be disjoint (no task ID in both).
- All IDs in `deps` and `softDeps` must exist in `items`.
- No duplicates within `deps`; no duplicates within `softDeps`.

## Readiness

- **Before**: Ready = leaf, todo, and `UnmetDeps(g, it) == nil` (all `deps` satisfied).
- **After**: Ready = leaf, todo, and all **hard** deps satisfied. `UnmetDeps` continues to consider only `deps` (hard). New helper or overload for "unmet hard deps" only if needed; existing `UnmetDeps` stays as unmet **hard** deps.
- Soft deps are never consulted for readiness. A task can be ready even if some soft deps are not done.

## Task selection order (unblocks-most)

1. Compute the set of **ready** task IDs (leaf, todo, all hard deps satisfied).
2. For each ready task `id`, compute **unblocks-most(id)** = number of **other** items that (a) reference `id` in `deps` or `softDeps`, and (b) have status not done (e.g. not `StatusDone` and not `StatusSkipped`).
3. Sort ready IDs by unblocks-most **descending** (higher count first), then by task ID ascending for a stable tie-break.
4. The runner (and any consumer that "picks next") takes the **first** element of this sorted list.

Example: Ready = [A, B, C]. Others depend on A (3), B (1), C (0). Order = A, B, C. So we run A next.

## Dependents and reverse lookups

- **Dependents(g, id)**: Today returns items that have `id` in `deps`. Extend to return items that have `id` in `deps` **or** `softDeps`. Output remains sorted, stable.
- Display of "who depends on me" (if any) shows only **hard** dependents in the UI; internal Dependents() and unblocks-most still use both hard and soft.

## Validation (summary)

- `softDeps` required (non-nil); use `[]` if none.
- For each item: no duplicate in `deps`; no duplicate in `softDeps`; no ID in both `deps` and `softDeps`; every ID in `deps` and `softDeps` exists in `items`.
- `depRationale`: each key must be in `deps` or `softDeps` (either is allowed).
- Cycle detection: unchanged, over **hard** deps only (`deps`).

## Display and editing: soft deps are under-the-hood only

- **No rendering of soft deps** in CLI or TUI: list, show, pick, and detail views display only **hard** deps (`deps`). Unmet-deps text, "deps satisfied," and dep lists refer to hard deps only. Soft deps are not shown.
- **No CLI/TUI editing of soft deps**: There are no commands or UI controls to add/remove soft deps. Users can set `softDeps` only by editing the plan file (JSON) directly or via code/agent that writes the plan.
- Soft deps still affect **ordering** (unblocks-most) and **Dependents** internally; they are just not visible or editable in the UI.

## Backward compatibility

- Existing plans without `softDeps`: decode as `nil` → treat as `[]` (empty). No change to readiness or ordering until soft deps are added.

## Non-goals

- Rendering or editing soft deps in CLI/TUI (soft deps are plan-only; visible only in the plan JSON or in code).
- Priority field or other ordering signals (can be added later).
- Soft-dep cycle detection or validation (soft cycles allowed).
- Changing parent/child or leaf rules; only dep semantics and ready-task ordering.

## Done criteria

- Plans can store `softDeps`; validation enforces mutual exclusivity and references.
- Readiness uses only hard deps.
- Ready task order is unblocks-most (desc) then ID (asc); runner and pick/list use this order.
- Dependents and depRationale updated as above; cycle detection hard-only.
- **No** CLI/TUI rendering or editing of soft deps (under-the-hood only).
- Tests and docs updated.
