package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jbonatakis/blackbird/internal/plan"
)

type pickRow struct {
	index  int
	id     string
	status plan.Status
	ready  string
	title  string
	detail string
}

type pickStats struct {
	total           int
	ready           int
	blockedOnDeps   int
	manualBlocked   int
	includeNonLeafs bool
}

func runPick(args []string) error {
	fs := flag.NewFlagSet("pick", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	includeNonLeaf := fs.Bool("include-non-leaf", false, "include non-leaf items")
	all := fs.Bool("all", false, "show all items in scope")
	blocked := fs.Bool("blocked", false, "show blocked items in scope")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "pick takes only flags (no positional args)"}
	}

	path := planPath()

	for {
		g, err := loadValidatedPlan(path)
		if err != nil {
			return err
		}

		rows, stats := pickRows(g, *includeNonLeaf, *all, *blocked)
		if len(rows) == 0 {
			printPickEmptyMessage(stats, *all, *blocked)
			return nil
		}

		printPickList(os.Stdout, rows, *all, *blocked)
		selectedID, err := promptPickSelection(rows)
		if err != nil {
			return err
		}
		if selectedID == "" {
			return nil
		}

		fmt.Fprintln(os.Stdout)
		if err := runShow(selectedID); err != nil {
			return err
		}

		action, err := promptPickAction()
		if err != nil {
			return err
		}

		switch action {
		case "back":
			continue
		case "exit":
			return nil
		case "in_progress":
			if err := runSetStatus(selectedID, string(plan.StatusInProgress)); err != nil {
				return err
			}
		case "done":
			if err := runSetStatus(selectedID, string(plan.StatusDone)); err != nil {
				return err
			}
		case "blocked":
			if err := runSetStatus(selectedID, string(plan.StatusBlocked)); err != nil {
				return err
			}
		}
	}
}

func pickRows(g plan.WorkGraph, includeNonLeaf bool, all bool, blocked bool) ([]pickRow, pickStats) {
	var ids []string
	if includeNonLeaf {
		ids = make([]string, 0, len(g.Items))
		for id := range g.Items {
			ids = append(ids, id)
		}
	} else {
		ids = leafIDs(g)
	}
	sort.Strings(ids)

	rows := make([]pickRow, 0, len(ids))
	stats := pickStats{total: len(ids), includeNonLeafs: includeNonLeaf}

	for _, id := range ids {
		it, ok := g.Items[id]
		if !ok {
			continue
		}

		unmet := plan.UnmetDeps(g, it)
		depsOK := len(unmet) == 0
		actionable := it.Status == plan.StatusTodo && depsOK
		isBlocked := !depsOK || it.Status == plan.StatusBlocked

		statusCounts(unmet, it.Status, depsOK, &stats)

		if !all && !blocked {
			if !actionable {
				continue
			}
		} else if blocked {
			if !isBlocked {
				continue
			}
		}

		if actionable {
			stats.ready++
		}

		readyLabel := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)

		var details string
		if len(unmet) > 0 {
			details = "unmet deps: " + strings.Join(unmet, ", ")
		} else if it.Status == plan.StatusBlocked {
			details = "manually blocked (deps satisfied)"
		}

		rows = append(rows, pickRow{
			index:  len(rows) + 1,
			id:     it.ID,
			status: it.Status,
			ready:  readyLabel,
			title:  it.Title,
			detail: details,
		})
	}

	return rows, stats
}

func statusCounts(unmet []string, status plan.Status, depsOK bool, stats *pickStats) {
	if stats == nil {
		return
	}
	if len(unmet) > 0 {
		stats.blockedOnDeps++
		return
	}
	if status == plan.StatusBlocked && depsOK {
		stats.manualBlocked++
	}
}

func printPickList(w io.Writer, rows []pickRow, all bool, blocked bool) {
	label := "Tasks:"
	if !all && !blocked {
		label = "Ready tasks:"
	}
	fmt.Fprintln(w, label)
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	for _, r := range rows {
		if r.detail != "" {
			fmt.Fprintf(tw, "%d)\t%s\t%s\t%s\t%s\t%s\n", r.index, r.id, r.status, r.ready, r.title, r.detail)
		} else {
			fmt.Fprintf(tw, "%d)\t%s\t%s\t%s\t%s\n", r.index, r.id, r.status, r.ready, r.title)
		}
	}
	_ = tw.Flush()
}

func promptPickSelection(rows []pickRow) (string, error) {
	for {
		line, err := promptLine("Select task (number or id, blank to exit)")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(line) == "" {
			return "", nil
		}
		if id, ok := pickSelectionFromInput(line, rows); ok {
			return id, nil
		}
		fmt.Fprintln(os.Stdout, "invalid selection; try again")
	}
}

func pickSelectionFromInput(input string, rows []pickRow) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}

	if n, err := parseInt(input); err == nil {
		if n >= 1 && n <= len(rows) {
			return rows[n-1].id, true
		}
	}

	for _, r := range rows {
		if r.id == input {
			return r.id, true
		}
	}

	return "", false
}

func promptPickAction() (string, error) {
	for {
		line, err := promptLine("Action ([i]n_progress, [d]one, [b]locked, back, exit)")
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "i", "in_progress", "in-progress", "in progress":
			return "in_progress", nil
		case "d", "done":
			return "done", nil
		case "b", "blocked":
			return "blocked", nil
		case "back":
			return "back", nil
		case "exit", "q", "quit":
			return "exit", nil
		default:
			fmt.Fprintln(os.Stdout, "invalid action; try again")
		}
	}
}

func printPickEmptyMessage(stats pickStats, all bool, blocked bool) {
	if stats.total == 0 {
		fmt.Fprintln(os.Stdout, "no tasks found (add tasks with `blackbird add`)")
		return
	}

	var parts []string
	if blocked {
		parts = append(parts, "0 blocked")
	} else {
		parts = append(parts, "0 ready")
	}
	if stats.blockedOnDeps > 0 {
		parts = append(parts, fmt.Sprintf("%d blocked on deps", stats.blockedOnDeps))
	}
	if stats.manualBlocked > 0 {
		parts = append(parts, fmt.Sprintf("%d manually blocked", stats.manualBlocked))
	}
	if len(parts) == 1 {
		if all {
			parts = append(parts, fmt.Sprintf("%d total", stats.total))
		} else {
			parts = append(parts, fmt.Sprintf("%d not ready", stats.total))
		}
	}

	scope := "leaf"
	if stats.includeNonLeafs {
		scope = "all"
	}

	fmt.Fprintf(os.Stdout, "%s in %s scope; try `blackbird list --blocked` or `blackbird show <id>` for details\n", strings.Join(parts, "; "), scope)
}
