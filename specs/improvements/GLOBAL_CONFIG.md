status: pending
---
name: global-config
overview: Add a global configuration system with a config file in the user's home directory (~/.blackbird/config.json), overridable by project-level config (.blackbird/config.json), for TUI refresh intervals and other shared settings.
todos:
  - id: config-spec-format
    content: Define config.json schema and precedence (global → project)
    status: pending
  - id: config-loader
    content: Implement config loader that merges global + project with override semantics
    status: pending
  - id: tui-intervals
    content: Wire TUI run/plan refresh intervals from config (with defaults)
    status: pending
  - id: docs-tests
    content: Document config in docs/CONFIGURATION.md and add tests
    status: pending
---

# Global Configuration System

## Goals

- **Single place for user defaults**: `~/.blackbird/config.json` for global preferences (e.g. TUI refresh rate, future options).
- **Project overrides**: Project-level `.blackbird/config.json` overrides global for that project only.
- **Explicit precedence**: Project config > global config > built-in defaults. No env vars in this spec for these options (env can be added later if needed).
- **Low surface area**: Start with a small set of keys (TUI intervals); add more as needed without changing precedence rules.

## Config locations and precedence

| Layer        | Location                    | Purpose                          |
|-------------|-----------------------------|----------------------------------|
| Built-in    | (code defaults)             | Defaults when no file exists     |
| Global      | `~/.blackbird/config.json`  | User-wide defaults               |
| Project     | `<project>/.blackbird/config.json` | Per-project overrides (optional) |

**Precedence**: For each key, use the first defined value in: project → global → built-in. Missing files or missing keys are skipped; invalid values should fall back to the next layer or built-in and optionally log a warning.

**Why `~/.blackbird/config.json` for global**: Same filename and schema as project config; only the root differs (home vs project). One format to parse and document; users can copy or symlink between home and project if desired.

## Schema (initial)

**Global**: `~/.blackbird/config.json` — single JSON object (same schema as project).

**Project**: `.blackbird/config.json` — single JSON object, same key set.

Suggested schema version and keys:

```json
{
  "schemaVersion": 1,
  "tui": {
    "runDataRefreshIntervalSeconds": 5,
    "planDataRefreshIntervalSeconds": 5
  }
}
```

- **`schemaVersion`**: Reserved for future breaking changes; loader should validate (e.g. support only `1` for now).
- **`tui.runDataRefreshIntervalSeconds`**: Seconds between run-data polls in the TUI (current constant: `runDataRefreshInterval` in `internal/tui/run_loader.go`). Minimum 1, maximum e.g. 300; invalid → fall back to next layer or default 5.
- **`tui.planDataRefreshIntervalSeconds`**: Seconds between plan-data polls (current: `planDataRefreshInterval` in `internal/tui/plan_loader.go`). Same bounds and default 5.

All keys optional. Omitted keys mean "use next layer or built-in default."

## Built-in defaults

| Key                                  | Default | Notes                    |
|--------------------------------------|---------|--------------------------|
| `tui.runDataRefreshIntervalSeconds`  | 5       | Current hardcoded value  |
| `tui.planDataRefreshIntervalSeconds` | 5       | Current hardcoded value  |

## Key code touchpoints

- **New package or internal pkg**: Add `internal/config` (or similar) with:
  - `LoadConfig(projectRoot string) (ResolvedConfig, error)`:
    - Read `~/.blackbird/config.json` (use `os.UserHomeDir()`; if home unknown, skip global).
    - Read `projectRoot/.blackbird/config.json` if present.
    - Merge with precedence above; validate and clamp intervals; return struct used by TUI/CLI.
  - Types: `GlobalConfig`, `ProjectConfig`, `ResolvedConfig` (or single struct with merged values).
- **TUI**: In `internal/tui`, where the program is started (e.g. `tea.NewProgram` or model init), call config loader once (with `os.Getwd()` or explicit project root) and pass resolved intervals into the model or into `RunDataRefreshCmd` / `PlanDataRefreshCmd` (or equivalent). Replace `runDataRefreshInterval` / `planDataRefreshInterval` constants with values from `ResolvedConfig`.
- **Existing project config**: `.blackbird/agent.json` stays as-is (agent selection only). No need to merge it into this schema; keep agent selection in its current file and type.
- **Docs**: Update `docs/CONFIGURATION.md` to describe `~/.blackbird/config.json` and `<project>/.blackbird/config.json`, precedence, and the initial keys. Update `docs/FILES_AND_STORAGE.md` to list both config paths.

## Edge cases and validation

- **Missing home**: If `os.UserHomeDir()` fails, skip global config and use built-in defaults (and optionally project config).
- **Invalid JSON**: Skip that file and use next layer; optionally warn to stderr or log.
- **Unknown keys**: Ignore (forward-compatible).
- **Invalid values**: e.g. `runDataRefreshIntervalSeconds: 0` or `-1` → clamp to allowed range or use default for that key; same for plan interval.
- **Project root**: Use working directory when launching TUI/CLI unless a future flag specifies a project root; then pass that into `LoadConfig(projectRoot)`.

## Future extensions (out of scope for initial spec)

- Env overrides (e.g. `BLACKBIRD_TUI_RUN_REFRESH_SEC`) with precedence env > project > global > built-in.
- More TUI options (e.g. theme, key bindings).
- CLI-wide options (e.g. default output format, logging).
- Local-only config (e.g. `.blackbird/config.local.json` gitignored) for secrets or machine-specific overrides.
