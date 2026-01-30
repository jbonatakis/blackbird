---
name: tui-home-screen
overview: Add a Home screen to the TUI and allow startup without a plan file.
---

# TUI Home Screen + Missing Plan Handling

## Goals
- Launch the TUI even when `blackbird.plan.json` is missing.
- Always start on a clean Home screen.
- Gate plan-dependent views/actions when no plan file exists.
- Reuse existing plan flows (generate/refine/edit) from the Home screen.

## Non-Goals
- Redesign plan/execution panes.
- Change the plan schema or file format.
- Auto-create a plan file on TUI startup.

## Product Decisions (from user)
- Only create `blackbird.plan.json` after explicit user action.
- Always default to the Home screen on launch.
- “Create plan (manual)” routes to existing plan flows (no new flow).
- Views that require a plan are inaccessible if it does not exist.

## UX Overview

### Home Screen (default view)
Centered, minimal layout with:
- Title + short tagline.
- Plan status line:
  - “No plan found” when missing.
  - “Plan found: <n> items, <ready> ready, <blocked> blocked” when present.
- Action list with single-key shortcuts.
- Disabled labels for actions that require a plan when none exists.

### Main View (existing)
- Unchanged plan tree + detail/execution panes.
- Accessed from Home when a plan exists.

## User Actions and Availability

When plan is missing:
- **Generate plan**: enabled (opens existing generate modal).
- **View plan**: disabled.
- **Refine plan**: disabled.
- **Execute**: disabled.
- **Quit**: enabled.

When plan exists:
- **View plan**: enabled (opens main view).
- **Generate plan**: enabled (may trigger overwrite confirm).
- **Refine plan**: enabled.
- **Execute**: enabled only when ready tasks exist.
- **Quit**: enabled.

## Keybindings (proposed)
- `h` toggle Home (from any view).
- `g` generate plan.
- `r` refine plan (only if plan exists).
- `v` view plan (only if plan exists).
- `e` execute (only if ready tasks exist).
- `ctrl+c` quit (consistent with existing behavior).

## Plan File Handling

### Startup
- If `blackbird.plan.json` is missing:
  - Initialize an in-memory empty graph via `plan.NewEmptyWorkGraph()`.
  - Track `planExists=false`.
  - Start on Home screen.

### Refresh
- Plan reload tick should:
  - Treat missing plan as non-error state.
  - Set `planExists=false` and keep empty in-memory graph.
  - If a plan becomes available, set `planExists=true` and refresh counts.

### Validation Errors
- If a plan exists but is invalid, show an error banner on Home with a short message and a prompt to regenerate or open the plan view for fixes.

## TUI State/Model Changes

Add fields to the TUI model:
- `viewMode` (enum): `Home`, `Main`
- `planExists` (bool)

Optional helper methods:
- `hasPlan()` and `canExecute()` for gating.

## Rendering Changes

### Home View Renderer
New file `internal/tui/home_view.go`:
- `RenderHomeView(m Model) string`
- Uses lipgloss for centered layout and muted/dim style for disabled actions.

### Main View
If `viewMode == Home`, render home view instead of the split panes.

### Bottom Bar
If `viewMode == Home`:
- Show only Home-relevant hints.
- Hide status counts if plan missing; otherwise include counts.

## Action Handling

### Key routing
- On `h`, toggle between Home/Main (Main only if plan exists).
- On Home, only allow actions if enabled.
- On Main, preserve existing key behavior.

### Generate flow
- Use current `g` flow and confirmation logic.
- On successful generate, `planExists=true` and move to Main view (optional, but recommended).

## Error Handling
- TUI start must not error when plan is missing.
- Plan refresh missing file is not fatal.
- Validation errors surfaced but do not crash the UI.

## Tests
- Start TUI without plan file: no error, `planExists=false`, Home view.
- Plan refresh when plan missing: remains Home, no crash.
- Home action gating:
  - `v` ignored when no plan.
  - `g` opens generate modal.
- Plan exists:
  - `v` opens main view.
  - `e` only enabled when ready tasks > 0.

## Dependencies/Notes
- Keep dependencies low; use existing Bubble Tea + lipgloss only.
- No new external assets.
