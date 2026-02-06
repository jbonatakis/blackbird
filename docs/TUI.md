# TUI overview

Running `blackbird` with no arguments launches the TUI. CLI commands like `blackbird plan`, `blackbird execute`, and `blackbird list` are unchanged.

## Layout

- **Left pane** — Plan tree with status and readiness labels.
- **Right pane** — Details or execution dashboard (toggle with `t`).
- **Bottom bar** — Action shortcuts and ready/blocked counts.
- **Home view** — Shows the current agent selection; press `c` to open the agent picker (selection persists to `.blackbird/agent.json`) and `s` to open Settings.
- **Settings view** — Table of config options with local/global/default/applied values and inline editing.

## Key bindings

| Key | Action |
|-----|--------|
| `up` / `down` or `j` / `k` | Move selection in the tree |
| `enter` or space | Expand/collapse parent items |
| `tab` | Switch focus between tree and detail panes |
| `f` | Cycle filters (all, ready, blocked) |
| `pgup` / `pgdown` | Scroll the detail pane |
| `t` | Switch details/execution tab |
| `g` | Plan generate |
| `r` | Plan refine |
| `e` | Execute ready tasks |
| `c` | Change agent (Home view) |
| `s` | Settings (Home view); set status for selected item (Main view) |
| `u` | Resume waiting task (when available) |
| `ctrl+c` | Quit |

**Settings View**
Open from Home with `s`. The Settings table shows `Option`, `Local`, `Global`, `Default`, and `Applied` columns. Local/global are editable; default/applied are read-only. Edits autosave to the selected layer and refresh applied values; save or load errors appear in the footer.

The table includes `Planning Max Auto-Refine Passes` (`planning.maxPlanAutoRefinePasses`), which controls bounded auto-refine during plan generation (`0` disables; clamped to `0`..`3`; default `1`).

Keys:
- `up` / `down` or `j` / `k` — Move row selection
- `left` / `right` — Move column selection
- `space` — Toggle boolean values in the Local/Global columns
- `enter` — Edit integer values in the Local/Global columns (digits only; `enter` saves, `esc` cancels)
- `backspace` / `delete` — Clear the selected Local/Global value (removes the key from that layer)
- `esc` or `h` — Return to Home
- `ctrl+c` — Quit

## Plan generate/refine @path lookup

The plan generate and plan refine modals support `@` file lookup inside their text areas (description/constraints/granularity and the refine change request). Type `@` to open the picker at the cursor, then keep typing to filter workspace paths. Use `up` / `down` to change selection, `enter` to insert the selected path (replacing the `@query` span), and `esc` to close without inserting. `tab` / `shift+tab` close the picker so focus can move between fields; `backspace` edits the query while the picker is open.

## Plan quality review (generate)

After plan generation, the review modal shows a quality summary derived from deterministic lint:
- Initial and final blocking/warning counts.
- Auto-refine pass outcome when at least one pass ran.
- Key findings excerpt (or `none`).

If blocking findings remain, the modal requires explicit override to save:
- Action `1` is labeled `Accept anyway`.
- Default selection is not accept (it defaults to `Revise`, or `Reject` if revisions are exhausted).
- Saving through override shows a warning that blocking findings were overridden.

**Review Checkpoints**
When `execution.stopAfterEachTask` is enabled, execution pauses after each task finishes and requires a decision before continuing. The TUI shows an “ACTION REQUIRED” banner and opens a review checkpoint modal with task details and a review summary (changed files, diffstat, snippets).

Actions:
- Approve and continue: record approval and run the next ready task.
- Approve and quit: record approval and stop execution.
- Request changes: open a multi-line change request and resume the same agent session for that task.
- Reject changes: mark the task failed and stop execution.

In the action chooser, use `up` / `down` or `1-4` to select and `enter` to confirm. When requesting changes, use `ctrl+s` or `ctrl+enter` to submit, `esc` to return to the action list, and `@` to open the file picker.

Limitations and errors:
- Review checkpoints are separate from `waiting_user` question prompts; `u` (resume) only applies to runs waiting on agent questions.
- `Request changes` requires provider resume support and a saved session reference; if resume is unsupported or missing, the decision will fail and the modal will report the error.
- Review summaries are best-effort; if git status/diff commands fail or time out, the modal shows an empty summary.
