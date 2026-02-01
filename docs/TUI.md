# TUI overview

Running `blackbird` with no arguments launches the TUI. CLI commands like `blackbird plan`, `blackbird execute`, and `blackbird list` are unchanged.

## Layout

- **Left pane** — Plan tree with status and readiness labels.
- **Right pane** — Details or execution dashboard (toggle with `t`).
- **Bottom bar** — Action shortcuts and ready/blocked counts.
- **Home view** — Shows the current agent selection; press `a` to open the agent picker (selection persists to `.blackbird/agent.json`).

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
| `a` | Change agent (Home view) |
| `u` | Resume waiting task (when available) |
| `s` | Set status for selected item |
| `ctrl+c` | Quit |

## Plan generate/refine @path lookup

The plan generate and plan refine modals support `@` file lookup inside their text areas (description/constraints/granularity and the refine change request). Type `@` to open the picker at the cursor, then keep typing to filter workspace paths. Use `up` / `down` to change selection, `enter` to insert the selected path (replacing the `@query` span), and `esc` to close without inserting. `tab` / `shift+tab` close the picker so focus can move between fields; `backspace` edits the query while the picker is open.
