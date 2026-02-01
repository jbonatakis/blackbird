# Agent runtime configuration

Blackbird invokes an external agent command for plan generation/refinement and execution. Configuration is environment-based:

| Variable | Description |
|----------|-------------|
| `BLACKBIRD_AGENT_PROVIDER` | `claude` or `codex` — selects the default command (defaults to `claude`). |
| `BLACKBIRD_AGENT_CMD` | Overrides the command entirely (runs via `sh -c`). |
| `BLACKBIRD_AGENT_STREAM` | Set to `1` to stream agent stdout/stderr live to the terminal. |
| `BLACKBIRD_AGENT_DEBUG` | Set to `1` to print the JSON request payload for debugging. |

The command must emit exactly one JSON object on stdout (either the full stdout or inside a fenced ```json block). Multiple objects or missing JSON fail fast.

When no environment provider is set, Blackbird falls back to the saved Home-screen agent selection in `.blackbird/agent.json`.
