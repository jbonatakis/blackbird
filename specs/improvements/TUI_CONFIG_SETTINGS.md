status: draft
---
name: tui-config-settings
overview: Add a TUI settings page to view and edit local and global config values alongside defaults and resolved values, with autosave and clear precedence indicators.
todos:
  - id: settings-view
    content: Add a Settings view accessible from Home with a config table and navigation
    status: pending
  - id: config-metadata
    content: Define a central list of config options (keys, types, defaults, validation) for the table
    status: pending
  - id: autosave
    content: Implement config editing with autosave, validation, and applied-source display
    status: pending
  - id: tests-docs
    content: Add tests and update docs/TUI.md + docs/CONFIGURATION.md
    status: pending
---

# TUI Config Settings

## Goals

- Provide a Settings page in the TUI that lists all available config options.
- Show values by source: Local (project), Global (home), Default, and Applied.
- Allow editing Local and Global values inline with autosave.
- Clearly indicate that Local overrides Global (and Global overrides Default).
- Reflect applied values using the same precedence and clamping as config resolution.

## Non-goals

- Editing schemaVersion (read-only / not part of the table).
- Supporting arbitrary/unknown config keys beyond the known option list.
- Adding new config keys as part of this work.
- Adding env var overrides or additional config layers.

## User experience

### Entry point

- Home view adds a new action: `[s] Settings`.
- Pressing `s` on the Home view opens the Settings page.
- `esc` or `h` returns to Home.

### Layout

Settings view is a full-screen table with fixed columns:

| Option | Local (project) | Global (home) | Default | Applied |

- **Option**: Stable config name (e.g. not `tui.planDataRefreshIntervalSeconds`, but something like `TUI Plan Refresh (seconds)`).
- **Local**: Value from `<project>/.blackbird/config.json`.
- **Global**: Value from `~/.blackbird/config.json`.
- **Default**: Built-in default from `config.DefaultResolvedConfig()`.
- **Applied**: Resolved value with source label (local/global/default).
- **Empty values**: Empty values should be displayed as a centered `-`

Header includes:
- A short precedence reminder: `Local > Global > Default`.
- File paths for Local and Global (resolved at runtime; show `N/A` if unavailable).

Footer includes:
- The description for the currently selected option (type, units, bounds).
- Any save/validation errors.

### Navigation and editing

- **Up/Down**: move row selection.
- **Left/Right**: move column selection.
- **Editable columns**: Local, Global only.
- **Enter**: toggle edit mode on the selected editable cell.
- **Esc**: cancel edit (revert to last saved value).
- **Delete** (or `backspace` on empty input): clear the value (unset) and autosave.

Editing behavior by type:

- **Bool** (`execution.stopAfterEachTask`):
  - `space` or `enter` toggles between true/false.
  - `delete` clears (unset).
- **Int** (refresh intervals):
  - In edit mode: digits to edit, backspace deletes, enter commits.
  - Reject non-numeric input.
  - Validate min/max on commit.

Autosave is triggered after every committed edit or clear.

### Precedence and applied display

- If Local is set, Applied = Local (clamped), and Local cell is highlighted as the source.
- Else if Global is set, Applied = Global (clamped), and Global cell is highlighted.
- Else Applied = Default.
- Applied column shows the value plus a source tag, e.g. `2s (local)`.
- If a stored value is out of range, show a warning indicator in its cell and show the clamped Applied value (e.g. `400 (clamped to 300)`).

## Config option inventory (initial)

| Key                                  | Type | Default | Bounds | Description                                |
| ------------------------------------ | ---- | ------- | ------ | ------------------------------------------ |
| `tui.runDataRefreshIntervalSeconds`  | int  | 5       | 1-300  | Run data refresh interval in seconds       |
| `tui.planDataRefreshIntervalSeconds` | int  | 5       | 1-300  | Plan data refresh interval in seconds      |
| `execution.stopAfterEachTask`        | bool | false   | n/a    | Pause execution for review after each task |

All options are optional. Missing values fall through to the next layer or default.

## Data model and resolution

- Use existing config loaders to read Local and Global raw configs:
  - Local: `config.LoadProjectConfig(projectRoot)`
  - Global: `config.LoadGlobalConfig()`
- Applied values must be computed with the same logic as `config.ResolveConfig` (including interval clamping).
- Add a small metadata registry in `internal/config` so the TUI has a single source of truth for:
  - Key path and display name
  - Type (bool/int)
  - Default value (from `DefaultResolvedConfig`)
  - Validation bounds
  - Description

This avoids reflection and keeps the option list explicit.

## Persistence and autosave

- Edits write to the appropriate config file immediately:
  - Local edits write `<project>/.blackbird/config.json` (create dir if missing).
  - Global edits write `~/.blackbird/config.json` (create dir if missing).
- Save should be atomic (temp file + rename) to match plan writes.
- Saved file should include `schemaVersion` and only the keys that are set.
- If all values in a layer are unset, remove the config file to keep a clean default state.

After any successful save:
- Recompute Applied values.
- Update the model's in-memory `config` (resolved config) so refresh timers use the new intervals.

## Error handling

- If a config file is invalid JSON or has an unsupported schema version, treat it as unset and show a warning in the footer. Saving will overwrite with valid JSON.
- If the Global config location is unavailable (no home dir), disable the Global column and show `N/A` in the header.
- If a save fails, show an error banner in the footer and keep the prior value in the table.

## Touchpoints

- `internal/tui/model.go`: add Settings view mode, state, and key handling for `s`/`esc`.
- `internal/tui/home_view.go`: add `[s] Settings` action line.
- `internal/tui/bottom_bar.go`: add Settings shortcut hint for Home (and optionally Main if desired).
- `internal/tui` new settings view/rendering + key handling.
- `internal/config`: add option metadata registry and atomic save helper for config layers.
- `docs/TUI.md`: document Settings view and key bindings.
- `docs/CONFIGURATION.md`: mention TUI settings editor and precedence reminder.

## Testing

- Config:
  - Save helper tests (create, update, clear removes file).
  - Applied value source tests (local/global/default selection + clamping).
- TUI:
  - Home view renders `[s] Settings`.
  - Settings view navigation (row/column moves) and edit/clear for bool/int.
  - Autosave triggers file writes and updates applied values.
  - Global-disabled state when home is unavailable.

## Decisions

- Settings is accessible only from the Home view.
- Invalid values in config files are shown raw with a warning indicator and red styling; Applied still uses the clamped/validated value.
