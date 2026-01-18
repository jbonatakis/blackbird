package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

type UsageError struct {
	Message string
}

func (e UsageError) Error() string { return e.Message }

func Usage() string {
	return `blackbird: structured project plan CLI

Usage:
  blackbird init
  blackbird validate
  blackbird list [--all] [--blocked] [--tree] [--features] [--status <status>]
  blackbird show <id>
  blackbird set-status <id> <status>

Statuses:
  todo | in_progress | blocked | done | skipped
`
}

func Run(args []string) error {
	if len(args) == 0 {
		return UsageError{Message: "missing command"}
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stdout, Usage())
		return nil
	case "init":
		if len(args) != 1 {
			return UsageError{Message: "init takes no arguments"}
		}
		return runInit()
	case "validate":
		if len(args) != 1 {
			return UsageError{Message: "validate takes no arguments"}
		}
		return runValidate()
	case "list":
		return runList(args[1:])
	case "show":
		if len(args) != 2 {
			return UsageError{Message: "show requires exactly 1 argument: <id>"}
		}
		return runShow(args[1])
	case "set-status":
		if len(args) != 3 {
			return UsageError{Message: "set-status requires exactly 2 arguments: <id> <status>"}
		}
		return runSetStatus(args[1], args[2])
	default:
		return UsageError{Message: fmt.Sprintf("unknown command: %q", args[0])}
	}
}

func planPath() string {
	wd, err := os.Getwd()
	if err != nil {
		// If this fails, other file ops will fail too; keep path deterministic.
		return plan.DefaultPlanFilename
	}
	return filepath.Join(wd, plan.DefaultPlanFilename)
}

func runInit() error {
	path := planPath()

	_, err := os.Stat(path)
	if err == nil {
		fmt.Fprintf(os.Stdout, "plan already exists: %s\n", path)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat plan file: %w", err)
	}

	g := plan.NewEmptyWorkGraph()
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "created plan: %s\n", path)
	return nil
}

func runValidate() error {
	path := planPath()

	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return err
	}

	errs := plan.Validate(g)
	if len(errs) == 0 {
		fmt.Fprintln(os.Stdout, "OK")
		return nil
	}

	fmt.Fprintf(os.Stdout, "invalid plan: %s\n", path)
	for _, e := range errs {
		fmt.Fprintf(os.Stdout, "- %s: %s\n", e.Path, e.Message)
	}
	return errors.New("validation failed")
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	all := fs.Bool("all", false, "show all leaf tasks")
	blocked := fs.Bool("blocked", false, "show blocked leaf tasks")
	tree := fs.Bool("tree", false, "show the full hierarchy tree")
	features := fs.Bool("features", false, "show top-level items only")
	statusStr := fs.String("status", "", "filter by status")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "list takes only flags (no positional args)"}
	}

	var statusFilter *plan.Status
	if *statusStr != "" {
		s, ok := parseStatus(*statusStr)
		if !ok {
			return UsageError{Message: fmt.Sprintf("invalid status %q", *statusStr)}
		}
		statusFilter = &s
	}

	path := planPath()
	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path)
	}

	if *tree {
		printTree(os.Stdout, g)
		return nil
	}

	ids := leafIDs(g)
	if *features {
		ids = rootIDs(g)
	}
	sort.Strings(ids)

	type row struct {
		id      string
		status  plan.Status
		ready   string
		title   string
		details string
	}

	var rows []row
	for _, id := range ids {
		it, ok := g.Items[id]
		if !ok {
			continue
		}

		unmet := plan.UnmetDeps(g, it)
		depsOK := len(unmet) == 0
		actionable := it.Status == plan.StatusTodo && depsOK

		isBlocked := !depsOK || it.Status == plan.StatusBlocked

		if statusFilter != nil && it.Status != *statusFilter {
			continue
		} else if statusFilter == nil {
			// Default selection behavior.
			if !*all && !*blocked {
				if !actionable {
					continue
				}
			} else if *blocked {
				if !isBlocked {
					continue
				}
			}
		}

		readyLabel := readinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)

		var details string
		if len(unmet) > 0 {
			details = "unmet deps: " + strings.Join(unmet, ", ")
		} else if it.Status == plan.StatusBlocked {
			details = "manually blocked (deps satisfied)"
		}

		rows = append(rows, row{
			id:      it.ID,
			status:  it.Status,
			ready:   readyLabel,
			title:   it.Title,
			details: details,
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	for _, r := range rows {
		if r.details != "" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.id, r.status, r.ready, r.title, r.details)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.id, r.status, r.ready, r.title)
		}
	}
	_ = w.Flush()
	return nil
}

func runShow(id string) error {
	path := planPath()
	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path)
	}

	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}

	unmet := plan.UnmetDeps(g, it)
	depsOK := len(unmet) == 0
	actionable := it.Status == plan.StatusTodo && depsOK
	dependents := plan.Dependents(g, id)

	fmt.Fprintf(os.Stdout, "ID: %s\n", it.ID)
	fmt.Fprintf(os.Stdout, "Title: %s\n", it.Title)
	fmt.Fprintf(os.Stdout, "Status: %s\n", it.Status)
	fmt.Fprintf(os.Stdout, "CreatedAt: %s\n", it.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(os.Stdout, "UpdatedAt: %s\n", it.UpdatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintln(os.Stdout)

	if it.Description != "" {
		fmt.Fprintln(os.Stdout, "Description:")
		fmt.Fprintf(os.Stdout, "%s\n\n", it.Description)
	}

	if len(it.AcceptanceCriteria) > 0 {
		fmt.Fprintln(os.Stdout, "Acceptance criteria:")
		for _, ac := range it.AcceptanceCriteria {
			fmt.Fprintf(os.Stdout, "- %s\n", ac)
		}
		fmt.Fprintln(os.Stdout)
	}

	if it.Notes != nil && *it.Notes != "" {
		fmt.Fprintln(os.Stdout, "Notes:")
		fmt.Fprintf(os.Stdout, "%s\n\n", *it.Notes)
	}

	fmt.Fprintln(os.Stdout, "Deps:")
	if len(it.Deps) == 0 {
		fmt.Fprintln(os.Stdout, "- (none)")
	} else {
		for _, depID := range it.Deps {
			dep, ok := g.Items[depID]
			if !ok {
				fmt.Fprintf(os.Stdout, "- %s [unknown]\n", depID)
				continue
			}
			fmt.Fprintf(os.Stdout, "- %s [%s] %s\n", depID, dep.Status, dep.Title)
		}
	}
	fmt.Fprintln(os.Stdout)

	fmt.Fprintln(os.Stdout, "Dependents:")
	if len(dependents) == 0 {
		fmt.Fprintln(os.Stdout, "- (none)")
	} else {
		for _, depID := range dependents {
			d, ok := g.Items[depID]
			if !ok {
				fmt.Fprintf(os.Stdout, "- %s [unknown]\n", depID)
				continue
			}
			fmt.Fprintf(os.Stdout, "- %s [%s] %s\n", depID, d.Status, d.Title)
		}
	}
	fmt.Fprintln(os.Stdout)

	fmt.Fprintln(os.Stdout, "Readiness:")
	if depsOK {
		fmt.Fprintln(os.Stdout, "- deps satisfied: yes")
	} else {
		fmt.Fprintf(os.Stdout, "- deps satisfied: no (unmet: %s)\n", strings.Join(unmet, ", "))
	}
	if it.Status == plan.StatusBlocked && depsOK {
		fmt.Fprintln(os.Stdout, "- manually blocked: yes (clear with `blackbird set-status "+it.ID+" todo`)")
	}
	fmt.Fprintf(os.Stdout, "- actionable now: %v\n", actionable)
	fmt.Fprintln(os.Stdout)

	fmt.Fprintln(os.Stdout, "Prompt:")
	if it.Prompt == "" {
		fmt.Fprintln(os.Stdout, "(empty)")
	} else {
		fmt.Fprintln(os.Stdout, it.Prompt)
	}
	fmt.Fprintln(os.Stdout)

	return nil
}

func runSetStatus(id string, statusStr string) error {
	s, ok := parseStatus(statusStr)
	if !ok {
		return UsageError{Message: fmt.Sprintf("invalid status %q", statusStr)}
	}

	path := planPath()
	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path)
	}

	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}

	it.Status = s
	it.UpdatedAt = time.Now().UTC()
	g.Items[id] = it

	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "updated %s status to %s\n", id, s)
	return nil
}

func parseStatus(s string) (plan.Status, bool) {
	switch plan.Status(s) {
	case plan.StatusTodo, plan.StatusInProgress, plan.StatusBlocked, plan.StatusDone, plan.StatusSkipped:
		return plan.Status(s), true
	default:
		return "", false
	}
}

func readinessLabel(status plan.Status, depsOK bool, manualBlocked bool) string {
	if status == plan.StatusDone {
		return "DONE"
	}
	if status == plan.StatusSkipped {
		return "SKIPPED"
	}
	if status == plan.StatusInProgress {
		return "IN_PROGRESS"
	}
	if !depsOK {
		return "BLOCKED"
	}
	if manualBlocked {
		return "BLOCKED"
	}
	if status == plan.StatusTodo {
		return "READY"
	}
	return ""
}

func leafIDs(g plan.WorkGraph) []string {
	out := make([]string, 0)
	for id, it := range g.Items {
		if len(it.ChildIDs) == 0 {
			out = append(out, id)
		}
	}
	return out
}

func rootIDs(g plan.WorkGraph) []string {
	out := make([]string, 0)
	for id, it := range g.Items {
		if it.ParentID == nil || *it.ParentID == "" {
			out = append(out, id)
		}
	}
	return out
}

func printTree(w io.Writer, g plan.WorkGraph) {
	roots := rootIDs(g)
	sort.Strings(roots)
	visited := map[string]bool{}
	for _, id := range roots {
		printTreeRec(w, g, id, "", visited)
	}
}

func printTreeRec(w io.Writer, g plan.WorkGraph, id string, indent string, visited map[string]bool) {
	if visited[id] {
		fmt.Fprintf(w, "%s%s [cycle]\n", indent, id)
		return
	}
	visited[id] = true

	it, ok := g.Items[id]
	if !ok {
		fmt.Fprintf(w, "%s%s [unknown]\n", indent, id)
		return
	}

	unmet := plan.UnmetDeps(g, it)
	depsOK := len(unmet) == 0
	readyLabel := readinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
	fmt.Fprintf(w, "%s%s\t%s\t%s\n", indent, it.ID, it.Status, readyLabel)

	children := append([]string{}, it.ChildIDs...)
	sort.Strings(children)
	for _, cid := range children {
		printTreeRec(w, g, cid, indent+"  ", visited)
	}
}
