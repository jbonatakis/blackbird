# TUI overview

Running `blackbird` with no arguments launches the TUI. CLI commands like `blackbird plan`, `blackbird execute`, and `blackbird list` are unchanged.

## Layout

- **Left pane** — Plan tree with status and readiness labels.
- **Right pane** — Details or execution dashboard (toggle with `t`).
- **Bottom bar** — Action shortcuts and ready/blocked counts.

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
| `u` | Resume waiting task (when available) |
| `s` | Set status for selected item |
| `ctrl+c` | Quit |
