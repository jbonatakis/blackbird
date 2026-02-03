# Configuration

Blackbird reads configuration from JSON files plus a small set of environment variables for the agent runtime.

## Config files

Blackbird loads two optional config files:

- Global: `~/.blackbird/config.json`
- Project: `<projectRoot>/.blackbird/config.json`

Precedence is per key: project config overrides global config, which overrides built-in defaults.

If a config file is missing, contains invalid JSON, or uses an unsupported `schemaVersion`, that layer is skipped.

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
  "memory": {
    "mode": "deterministic",
    "proxy": {
      "listenAddr": "127.0.0.1:8080",
      "upstreamURL": "https://api.openai.com",
      "chatGPTUpstreamURL": "https://chatgpt.com",
      "lossless": true
    },
    "retention": {
      "traceRetentionDays": 14,
      "traceMaxSizeMB": 512
    },
    "budgets": {
      "totalTokens": 1200,
      "decisionsTokens": 200,
      "constraintsTokens": 150,
      "implementedTokens": 300,
      "openThreadsTokens": 150,
      "artifactPointersTokens": 100
    }
  }
}
```

Defaults:

- `schemaVersion`: `1`
- `tui.runDataRefreshIntervalSeconds`: `5`
- `tui.planDataRefreshIntervalSeconds`: `5`
- `memory.mode`: `deterministic`
- `memory.proxy.listenAddr`: `127.0.0.1:8080`
- `memory.proxy.upstreamURL`: `https://api.openai.com`
- `memory.proxy.chatGPTUpstreamURL`: `https://chatgpt.com`
- `memory.proxy.lossless`: `true`
- `memory.retention.traceRetentionDays`: `14`
- `memory.retention.traceMaxSizeMB`: `512`
- `memory.budgets.totalTokens`: `1200`
- `memory.budgets.decisionsTokens`: `200`
- `memory.budgets.constraintsTokens`: `150`
- `memory.budgets.implementedTokens`: `300`
- `memory.budgets.openThreadsTokens`: `150`
- `memory.budgets.artifactPointersTokens`: `100`

Interval values are clamped to a minimum of `1` and a maximum of `300` seconds.

Memory retention values are clamped to `1-3650` days and `1-102400` MB. Memory budget values are clamped to `0-20000` tokens.

## Memory configuration

Memory is currently available for the Codex provider only. Claude ignores memory settings.

### Mode

`memory.mode` accepts: `off`, `passthrough`, `deterministic`, `local`, `provider` (case-insensitive). In the current implementation any mode other than `off` enables memory for supported providers; the string is preserved for future mode-specific behavior.

### Proxy behavior

When memory is enabled and the provider is Codex, Blackbird starts a local reverse proxy (see `memory.proxy.listenAddr`) when running the TUI or executing/resuming tasks. The agent is pointed at the proxy via `OPENAI_BASE_URL` / `OPENAI_API_BASE`, and `OPENAI_DEFAULT_HEADERS` injects `X-Blackbird-Session-Id`, `X-Blackbird-Task-Id`, and `X-Blackbird-Run-Id` so traces can be attributed.

The proxy forwards requests to the configured upstreams (`memory.proxy.upstreamURL` for API traffic and `memory.proxy.chatGPTUpstreamURL` for ChatGPT traffic), rewrites paths as needed, strips hop-by-hop headers, and records a trace WAL under `.blackbird/memory/trace/`.

If `memory.proxy.lossless` is `false`, request/response bodies are not stored (privacy mode). Sensitive headers are always redacted in trace files.

### Budgets

`memory.budgets` controls how many tokens are allotted to each section when building memory context packs (`blackbird mem context` and execution context packs). `totalTokens` caps the overall size; the remaining fields limit their respective sections.

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
