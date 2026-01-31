# Lipgloss / TUI layout learnings

Notes from building and fixing the blackbird TUI (charmbracelet/lipgloss, bubbletea).

## Height and borders

- **Lipgloss applies `Height(N)` to the inner block, then adds the border.** So a pane with `Height(availableHeight)` and a border has **total height = availableHeight + 2** (top border + content + bottom border).
- **Total view height:** main content (pane area) + newline + bottom bar = `(availableHeight+2) + 2`. To stay under terminal height, use `availableHeight = windowHeight - 5` so total = `windowHeight - 1`.
- **Exact-height output:** Rendering exactly `windowHeight` lines can cause first-line redraw bugs in some terminals. Keeping output one line short (`windowHeight - 1`) avoids the top border being cut off.

## Width and borders

- **Each bordered pane’s rendered width = content width + 2** (left border + content + right border).
- **Split for two panes:** To fit both panes on screen, `(leftWidth+2) + (rightWidth+2) = windowWidth`, so **leftWidth + rightWidth = windowWidth - 4**. Reserve 4 for the two borders, not 1 for a “gap”.

## Custom top border (e.g. title in border)

- **Do not replace runes by index in a lipgloss-rendered line.** The line includes ANSI escape codes; replacing runes corrupts those codes and can shorten the displayed line so the corner no longer aligns with the side.
- **Rebuild the top line** with the correct **display width**: build `"╭ " + title + " " + "─"*n + "╮"` and use the **first content line’s width** (from the already-rendered pane) as the target, then pad with middle dashes if the rebuilt line is short. Use `lipgloss.Width()` for display width (strips ANSI).

## JoinHorizontal

- **Height = max of the two blocks.** The shorter block is padded (at top or bottom depending on position). So if the two panes have different content heights, the joined result height can change (e.g. when switching Details vs Execution), which can make the bottom bar “jump”.
- **No gap between blocks** — they are placed side by side; line width is the sum of the two block widths.

## Viewport / detail pane

- **Viewport height** should match the pane’s **content** height (the `Height` passed to `renderPane`), i.e. the space inside the border. When passing a pane model, `model.windowHeight` is that content height; use it directly for the viewport height.
- **Scroll position:** Resetting `detailOffset` on every selection change causes the detail view to “jump” to the top when changing tasks. Only reset on tab switch or filter change if you want to preserve scroll position when moving between tasks.

## Testing

- **Placeholder / home view:** If `availableHeight = windowHeight - 5`, you need `windowHeight >= 6` so `availableHeight >= 1` and at least one content line is shown; with `windowHeight: 2`, only the bar is rendered.
- **`detailPageSize()`** should return the same height as the viewport (e.g. `windowHeight - 5` when that is the pane content height) so pgup/pgdown scroll by one viewport.
