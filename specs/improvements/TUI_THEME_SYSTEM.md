# TUI Theme System (Semantic Tokens + Registry)
status: proposed

## Purpose

The current TUI styling approach relies on inline `lipgloss.Color("...")` literals spread across many renderers and modals. This makes theme changes repetitive and brittle.

This spec introduces a centralized theme system so components use semantic style tokens instead of hardcoded color values.

## Goals

1. Centralize TUI color definitions in one place.
2. Make render code consume semantic tokens (for example `Accent`, `Danger`, `StatusReady`) instead of raw color IDs.
3. Support theme switching via config without touching renderer code.
4. Keep initial implementation scoped to color tokens only.
5. Preserve current visual output exactly under the `blackbird` theme.
6. Keep the design extensible for future non-color style tokens (bold, italic, underline, border variants).
7. Apply theme changes live in the running TUI session (no restart required).

## Non-Goals

1. Supporting terminal font family or font size (terminal-controlled, not app-controlled).
2. Full custom user-defined token overrides in v1.
3. Redesigning layout/spacing of current TUI screens.
4. Changing behavior/state logic unrelated to rendering.

## Current Problem Summary

1. Color values are repeated inline across view files (`home`, `tree`, `execution`, and modal renderers).
2. Similar semantic meanings are duplicated (for example status colors in both home and tree views).
3. Tests have some direct coupling to literal color IDs.
4. Config does not currently include theme selection.

## High-Level Design

### 1. Theme model

Add a TUI theme model in `internal/tui/theme.go`:

```go
type Theme struct {
	ID     string
	Colors ColorTokens
	// Future: Text, Borders, Spacing, Icons, etc.
}

type ColorTokens struct {
	TextPrimary   lipgloss.Color
	TextMuted     lipgloss.Color
	TextOnAccent  lipgloss.Color
	Accent        lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Danger        lipgloss.Color
	Surface       lipgloss.Color
	SurfaceActive lipgloss.Color
	SurfaceMuted  lipgloss.Color
	Border        lipgloss.Color
	BorderActive  lipgloss.Color
}
```

### 2. Theme registry

Define built-in themes in `internal/tui/theme_registry.go`:

1. `blackbird` (matches current palette exactly).
2. `high-contrast` (clearly distinct built-in theme).

Requirement: ship two selectable built-in themes in v1 so users can switch themes immediately.

Provide deterministic lookup:

```go
func ResolveTheme(themeID string) Theme
```

If theme ID is unknown, resolve using precedence with invalid-layer skip (local -> global -> `blackbird`) and emit a warning in settings/footer.

### 3. Semantic style helpers

Add helper methods in `internal/tui/theme_styles.go` so renderers use names, not color literals:

1. `StatusStyle(readiness string) lipgloss.Style`
2. `RunStatusStyle(status execution.RunStatus) lipgloss.Style`
3. `ModalStyles()`, `PaneStyles(active bool)`, `ButtonStyles(focused bool, disabled bool)`, etc.

Layout concerns (padding, width, height) stay local in renderer files. Semantic color/style concerns move to theme helpers.

### 4. Model integration

Store resolved theme on `Model`:

```go
type Model struct {
	...
	theme Theme
}
```

Initialize from config in startup path and refresh when settings/config are reloaded.
When theme config changes via Settings, update `Model.theme` immediately and redraw in the same session.

## Configuration

Add config key:

- `tui.theme` (string), default `"blackbird"`.

Behavior:

1. Precedence: project > global > built-in default (`blackbird`), with invalid-layer skip for unknown values.
2. Unknown value does not terminate loading; it emits a non-fatal warning and falls back to the next valid value in precedence order.
3. Settings page displays and edits this value like other options.
4. Settings should show selectable built-in theme IDs (`blackbird`, `high-contrast`) so switching does not require manual file edits.
5. Theme changes in Settings must apply without requiring the user to restart the TUI.

## Settings Model For Categorical Values

Current Settings editing is int/bool-focused. To support `tui.theme` and future categorical settings, add a first-class categorical option type instead of hardcoded theme-specific logic.

### Data model updates

1. Extend config option metadata with a categorical type and allowed values list.
   - Example direction:
     - `OptionTypeEnum` (or `OptionTypeStringChoice`)
     - `AllowedValues []string`
     - `DefaultString string`
2. Extend raw/applied option payloads to carry string values.
   - Example direction:
     - `RawOptionValue{String *string}` in addition to `Int` and `Bool`.
3. Add `tui.theme` to `RawTUI` and `ResolvedTUI`.
4. Add `tui.theme` to settings option registry with allowed values from the built-in theme ID list sorted alphabetically.

### Interaction model in Settings

For categorical cells (Local/Global columns):

1. `space` or `enter`: cycle to next allowed value (wrap-around).
2. `delete`/`backspace` on empty edit: clear/unset the layer value (same as other settings).
3. Footer/help text should show allowed values for the selected categorical option.
4. Categorical options do not enter free-text edit mode in v1.

For `tui.theme` specifically:

1. Apply a small debounce when cycling choices to avoid rapid visual churn.
2. Debounce duration: **500ms** after the last cycle key press.
3. Persist/apply behavior:
   - after debounce expires, save the selected theme value and apply it live,
   - each additional cycle key press before expiry resets the timer.

This supports `n` theme IDs without adding a custom picker modal. If `n` grows large later, a dedicated chooser can be added without changing the config model.

### Validation/fallback behavior

1. If config contains an unknown categorical value, report a settings warning and skip that layer when resolving applied value.
2. Theme fallback chain is:
   - valid local value if present,
   - else valid global value if present,
   - else `blackbird`.
3. Unknown values must not crash rendering; they should behave like any other invalid setting input.
4. Settings table displays invalid user-provided raw values (for local/global cells) directly in the cell, with warning styling:
   - red background on the cell,
   - foreground/text color chosen for clear contrast and readability over red.
5. Invalid raw display values are truncated to a hard cap of 20 characters and suffixed with `...` when truncated.
   - This is display-only truncation; full stored config value remains unchanged on disk.
6. Theme resolution still falls back to `blackbird` at runtime as a final safety net.

## Built-in Theme Ordering

Built-in theme IDs are presented and cycled in alphabetical order. This ordering must be deterministic across registry lookup, settings rendering, and cycling behavior.

## External Config Changes

Theme/config changes made outside the TUI should be applied automatically after they are saved to disk.

Required behavior:

1. Add a lightweight periodic config refresh command while the TUI is running.
2. Use the existing run-data refresh interval (`tui.runDataRefreshIntervalSeconds`) for this polling loop (no new interval setting in v1).
3. On refresh, reload resolved config and recompute `Model.theme`.
4. If theme changed externally, update all views immediately (including open modals) with no restart.
5. If Settings view is open, refresh displayed layer/applied values while preserving current selection/column.
6. If a text-edit cell is active for non-theme settings, do not overwrite in-progress typed input; apply refreshed layer/applied data after edit commit/cancel.

No strict latency SLA is defined in v1 beyond the configured poll cadence.

## Migration Plan (Incremental)

### Phase 1: Foundation (no visual changes)

1. Add theme structs, tokens, registry, resolver.
2. Add `tui.theme` config plumbing.
3. Wire `Model.theme` and default resolution.
4. Add tests for resolver, fallback behavior, alphabetical ordering, and switching between at least two built-in themes.

### Phase 2: Core views

Migrate these first:

1. `internal/tui/model.go` pane border/title colors.
2. `internal/tui/tree_view.go` readiness + reviewing indicator styles.
3. `internal/tui/home_view.go` status summary and action styles.
4. `internal/tui/execution_view.go` run status styles.

### Phase 3: Shared UI primitives and modals

1. `internal/tui/file_picker_render.go`
2. `internal/tui/actions.go`
3. All modal renderers:
   - `plan_generate_modal.go`
   - `plan_refine_modal.go`
   - `plan_review_modal.go`
   - `review_checkpoint_modal.go`
   - `parent_review_modal.go`
   - `agent_question_modal.go`
   - `agent_selection_modal.go`

### Phase 4: Cleanup and enforcement

1. Remove remaining direct `lipgloss.Color("...")` usage outside theme files.
2. Refactor tests that assert raw IDs to semantic expectations.
3. Optionally add a guard test that fails when raw color literals are added outside theme modules.

## Testing Requirements

1. Unit tests for theme resolver:
   - valid ID resolution
   - unknown ID fallback with invalid-layer skip
   - deterministic token values for each built-in theme
   - switching from one built-in theme to another updates resolved tokens
   - built-in theme IDs are returned in alphabetical order
2. Config tests for `tui.theme` load/resolve/precedence.
   - include categorical validation for unknown theme IDs
   - include allowed-value roundtrip via `SaveConfigValues` / `ResolveSettings`
   - include invalid local + valid global fallback to global
   - include invalid local + invalid global fallback to `blackbird`
3. TUI rendering regression tests for:
   - no behavior regressions in status/readiness highlighting
   - modal border/outcome semantics
   - changing `tui.theme` from Settings updates rendered styles in the same session (no restart)
   - while any modal is open, theme change re-renders that modal immediately
   - invalid raw categorical values render with warning styling and truncated display
4. Settings interaction tests for categorical options:
   - `space`/`enter` cycles through `n` theme choices with wrap-around in alphabetical order
   - clear/unset restores precedence fallback behavior
   - footer displays allowed values for selected categorical option
   - categorical options never enter free-text edit mode in v1
   - `tui.theme` cycling uses a 500ms debounce before save/apply; rapid key presses reset the timer
5. External-config refresh tests:
   - editing config file externally and saving triggers in-session theme update on next refresh cycle
6. Warning-style tests:
   - invalid raw categorical values render in red-background warning cells with readable foreground contrast
   - invalid raw value display is truncated to 20 chars + `...` when exceeded
7. Update tests that currently assert direct color IDs to assert semantic branch behavior instead.

## Risks and Mitigations

1. Risk: style drift during migration.
   - Mitigation: keep default token values equal to current literals and migrate incrementally.
2. Risk: duplicated helper logic between renderers and theme helpers.
   - Mitigation: define clear helper boundaries and move shared status mappings into one place.
3. Risk: tests become brittle due to ANSI comparisons.
   - Mitigation: keep semantic tests focused on tokens/branches; strip ANSI in content-focused tests.

## Future Extension (Post-v1)

The model is intentionally expandable:

1. Add text emphasis tokens (`Bold`, `Italic`, `Underline`) and component style variants.
2. Add border/icon tokens by semantic role.
3. Add optional user token overrides once built-in theme switching is stable.

Terminal limitation note:

- Font family and font size are not controlled by the app; they are terminal settings.

## Done Criteria

1. No renderer depends on raw `lipgloss.Color("...")` literals outside theme modules.
2. Theme changes occur by selecting a different theme ID, without renderer code changes.
3. `tui.theme` is configurable with correct precedence and fallback (`local valid` -> `global valid` -> `blackbird`).
4. At least two built-in themes are shipped and selectable in TUI settings (`blackbird` + one distinct alternative).
5. `blackbird` theme matches current visuals exactly (no intentional deltas).
6. Built-in theme ordering is alphabetical and deterministic.
7. Changing theme in TUI Settings takes effect immediately in the running session (no restart), including open modals/panes.
8. External config file changes are picked up in-session on refresh and re-render with no restart.
9. Test suite covers resolver/config/rendering semantics for the theme system, including built-in theme switching, invalid-value fallback chain, and live in-session apply.
