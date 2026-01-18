TITLE
Phase 1 — Code Review (through M3)

DATE
2026-01-18

SCOPE
Manual edit CLI + mutation layer (M1–M3), readiness/deps behavior (M2).

FINDINGS (ORDERED BY SEVERITY)
1) Medium: Failed dep edits still mutate updatedAt, so a caller that keeps working
   with the in-memory graph could persist a timestamp change even though the edit
   errored. Consider restoring the prior UpdatedAt on rollback.
   - internal/plan/mutate.go:170
   - internal/plan/mutate.go:190
   - internal/plan/mutate.go:230
   - internal/plan/mutate.go:273

2) Medium: `blackbird delete --force` can remove dependency edges but the CLI
   only reports the deleted count, so users may not realize other tasks were
   altered; consider printing DetachedIDs or a summary.
   - internal/cli/manual.go:271
   - internal/cli/manual.go:285
   - internal/plan/mutate.go:284

3) Low: `parentCycleIfMove` walks parent pointers without a visited guard; if an
   invalid plan slips past validation, this can loop indefinitely. A simple
   visited set would make it robust.
   - internal/plan/mutate.go:381

4) Low: DeleteItem can append the same dependent ID multiple times when a
   remaining node depended on multiple deleted nodes; this now surfaces as
   duplicate entries in `detached deps from:` output. Consider deduping before
   sorting/printing.
   - internal/plan/mutate.go:333
   - internal/cli/manual.go:285

TEST GAPS
- No CLI-level tests for add/edit/delete/move/deps workflows (happy + failure).
- No tests asserting failed dep edits leave UpdatedAt unchanged.
