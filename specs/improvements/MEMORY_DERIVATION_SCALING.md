# Memory derivation scaling (WAL replay)

## Current behavior
- After each task run, Blackbird replays the **entire** trace WAL to canonicalize logs.
- If an artifact store already exists, only the **current run** is processed into artifacts, but canonicalization still scans all prior WAL events.
- The memory index is **rebuilt from the full artifact store** on every task finish.

## Impact of doing nothing
- **O(N) WAL replay per task** where N = total events so far; total work grows roughly O(N^2) across long sessions.
- Increasing latency between tasks as WAL grows; the delay is paid on every task boundary.
- More CPU and disk I/O (read + JSON parse + index rebuild), which may become noticeable for large sessions or frequent runs.

## Options to address
1) **Incremental WAL replay**
   - Track a cursor (byte offset or last event timestamp) and only replay new entries.
   - Store cursor in `.blackbird/memory/trace/` metadata.

2) **Per-run WAL files**
   - Write WAL events to a file per run so derivation can replay only that run's file.
   - Keep a session-level WAL only for debugging/forensics if needed.

3) **Incremental index updates**
   - Insert or upsert only new artifacts into the SQLite index.
   - Avoid full rebuild; optionally compact/rebuild on a schedule.

4) **Background derivation**
   - Run derivation asynchronously after a task finishes, allowing the next task to start immediately.
   - Risk: next task may not see the latest artifacts unless it waits or has a freshness check.

## Decision point: missing IDs when headers are ignored
When the agent client ignores `OPENAI_DEFAULT_HEADERS`, WAL entries lack `session_id` / `task_id` / `run_id`.
Canonicalization then yields zero logs, so no canonical logs or artifact/index updates occur.

### Options
1) **Proxy default IDs (fallback stamping)**
   - Execution sets active `session/task/run` IDs on the proxy at run start.
   - Proxy fills missing IDs when headers are absent.
   - Risk: misattribution if multiple runs share a proxy concurrently.

2) **Per-run WAL files under session/task**
   - Execution sets active context on the proxy; proxy writes to
     `.blackbird/memory/trace/<session>/<task>/<run>.wal`.
   - Makes derivation per-run without replaying the full WAL.
   - Still requires active context (the proxy can’t infer IDs on its own).

3) **Strict header enforcement**
   - Fail fast or warn if `X-Blackbird-*` headers are missing.
   - Keeps attribution correct but breaks clients that don’t honor default headers.

4) **Agent command wrapper**
   - Wrap the Codex client to inject headers explicitly (if supported).
   - Avoids proxy changes, but depends on the client’s CLI/SDK capabilities.

## Notes
- Options can be combined: e.g., (1) incremental WAL + (3) incremental index.
- If we keep full WAL replay, we should cap WAL size aggressively or accept increasing latency.
