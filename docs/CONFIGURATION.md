# Configuration

Blackbird reads configuration from JSON files plus a small set of environment variables for the agent runtime.

## Config files

Blackbird loads two optional config files:

- Global: `~/.blackbird/config.json`
- Project: `<projectRoot>/.blackbird/config.json`

Precedence is per key: project config overrides global config, which overrides built-in defaults.

If a config file is missing, contains invalid JSON, or uses an unsupported `schemaVersion`, that layer is skipped.

**Settings Editor**
The Home screen includes a Settings view (`s`) in the TUI that edits the same config files. Local edits write to `<projectRoot>/.blackbird/config.json`; global edits write to `~/.blackbird/config.json`. Changes autosave per key and the Settings table shows the resolved applied value plus its source.

Reminder: precedence is per key — project overrides global, which overrides defaults. Clearing a value in Settings removes it from that layer so the next layer takes effect.

## Config schema

Current schema version: `1`.

Supported keys:

```json
{
  "schemaVersion": 1,
  "tui": {
    "runDataRefreshIntervalSeconds": 5,
    "planDataRefreshIntervalSeconds": 5
  },
  "planning": {
    "maxPlanAutoRefinePasses": 1
  },
  "execution": {
    "stopAfterEachTask": false
  }
}
```

Defaults:

- `schemaVersion`: `1`
- `tui.runDataRefreshIntervalSeconds`: `5`
- `tui.planDataRefreshIntervalSeconds`: `5`
- `planning.maxPlanAutoRefinePasses`: `1`
- `execution.stopAfterEachTask`: `false`

Interval values are clamped to a minimum of `1` and a maximum of `300` seconds.

`planning.maxPlanAutoRefinePasses` controls bounded auto-refine during `blackbird plan generate` (CLI and TUI):
- `0` disables auto-refine passes.
- `1` (default) allows one automatic pass.
- Values are clamped to `0`..`3`.

In TUI Settings, this appears as `Planning Max Auto-Refine Passes` and follows the same local/global/default/applied precedence and warning behavior as other integer options.

`execution.stopAfterEachTask` uses the same per-key precedence rules (project config overrides global, which overrides defaults). When enabled, execution pauses after each completed run to request a review decision.

## Agent runtime configuration

Blackbird invokes an external agent command for plan generation/refinement and execution. Configuration is environment-based:

| Variable                   | Description                                                               |
| -------------------------- | ------------------------------------------------------------------------- |
| `BLACKBIRD_AGENT_PROVIDER` | `claude` or `codex` — selects the default command (defaults to `claude`). |
| `BLACKBIRD_AGENT_CMD`      | Overrides the command entirely (runs via `sh -c`).                        |
| `BLACKBIRD_AGENT_STREAM`   | Set to `1` to stream agent stdout/stderr live to the terminal.            |
| `BLACKBIRD_AGENT_DEBUG`    | Set to `1` to print the JSON request payload for debugging.               |

The command must emit exactly one JSON object on stdout (either the full stdout or inside a fenced ```json block). Multiple objects or missing JSON fail fast.

When no environment provider is set, Blackbird falls back to the saved Home-screen agent selection in `.blackbird/agent.json`.
