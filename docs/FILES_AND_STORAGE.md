# Files and storage

| Path | Description |
|------|-------------|
| `blackbird.plan.json` | Plan file (repo root). |
| `~/.blackbird/config.json` | Global configuration file. |
| `<project>/.blackbird/config.json` | Project configuration file. |
| `.blackbird/agent.json` | Selected agent runtime (Home screen agent selection). |
| `.blackbird/runs/<taskID>/<runID>.json` | Run records. |
| `.blackbird/snapshot.md` | Optional snapshot file. Fallbacks to `OVERVIEW.md`, then `README.md` if missing. |
| `.blackbird/memory/` | Memory root directory (session metadata, trace WALs, canonical logs, artifacts, index). |
| `.blackbird/memory/session.json` | Memory session metadata (session id and goal). |
| `.blackbird/memory/trace/trace.wal` | Active trace WAL from the memory proxy; rotated files use `trace-<timestamp>.wal`. |
| `.blackbird/memory/canonical/` | Canonicalized logs for runs (`<runID>.json`, plus `canonical.json` when no run id is set). |
| `.blackbird/memory/artifacts.db` | Artifact store (JSON) derived from canonical logs. |
| `.blackbird/memory/index.db` | SQLite index used by `blackbird mem search`. |
