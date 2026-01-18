package cli

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

var promptReader = bufio.NewReader(os.Stdin)

func setPromptReader(r io.Reader) {
	promptReader = bufio.NewReader(r)
}

type multiStringFlag []string

func (m *multiStringFlag) String() string { return strings.Join(*m, ", ") }
func (m *multiStringFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func runAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "stable id (optional; will be generated if empty)")
	title := fs.String("title", "", "title (required; prompt if missing)")
	description := fs.String("description", "", "description")
	prompt := fs.String("prompt", "", "canonical prompt")
	notes := fs.String("notes", "", "notes")
	parent := fs.String("parent", "root", "parent id or 'root'")
	indexStr := fs.String("index", "", "insert index within parent's childIds (optional)")
	var ac multiStringFlag
	fs.Var(&ac, "ac", "acceptance criteria (repeatable)")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "add takes only flags (no positional args)"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	if strings.TrimSpace(*title) == "" {
		v, err := promptLine("Title")
		if err != nil {
			return err
		}
		*title = v
	}
	if strings.TrimSpace(*title) == "" {
		return UsageError{Message: "title is required (use --title or enter it interactively)"}
	}

	newID := strings.TrimSpace(*id)
	if newID == "" {
		newID = generateID(g)
	}
	if _, ok := g.Items[newID]; ok {
		return UsageError{Message: fmt.Sprintf("id already exists: %q", newID)}
	}

	now := time.Now().UTC()
	it := plan.WorkItem{
		ID:                 newID,
		Title:              strings.TrimSpace(*title),
		Description:        *description,
		AcceptanceCriteria: []string(ac),
		Prompt:             *prompt,
		ParentID:           nil,        // set by plan.AddItem
		ChildIDs:           []string{}, // required
		Deps:               []string{}, // required
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if strings.TrimSpace(*notes) != "" {
		n := *notes
		it.Notes = &n
	}

	var parentID *string
	if *parent != "" && *parent != "root" {
		p := *parent
		parentID = &p
	}

	var idx *int
	if strings.TrimSpace(*indexStr) != "" {
		n, err := parseInt(*indexStr)
		if err != nil {
			return UsageError{Message: err.Error()}
		}
		idx = &n
	}

	if err := plan.AddItem(&g, it, parentID, idx, now); err != nil {
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "added %s\n", newID)
	return nil
}

func runEdit(id string, args []string) error {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "title")
	description := fs.String("description", "", "description")
	clearDescription := fs.Bool("clear-description", false, "clear description field")
	prompt := fs.String("prompt", "", "canonical prompt")
	clearPrompt := fs.Bool("clear-prompt", false, "clear prompt field")
	notes := fs.String("notes", "", "notes")
	clearNotes := fs.Bool("clear-notes", false, "clear notes field")
	acClear := fs.Bool("ac-clear", false, "clear acceptance criteria")
	var ac multiStringFlag
	fs.Var(&ac, "ac", "acceptance criteria (repeatable; replaces full list when provided)")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "edit takes only flags (no positional args after <id>)"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}

	// If no flags were provided, do a minimal interactive edit.
	noEdits := *title == "" && *description == "" && !*clearDescription && *prompt == "" && !*clearPrompt && *notes == "" && !*clearNotes && !*acClear && len(ac) == 0
	if noEdits {
		v, err := promptLineDefault("Title", it.Title)
		if err != nil {
			return err
		}
		if strings.TrimSpace(v) != "" {
			*title = v
		}
		v, err = promptLineDefault("Description (blank keeps current)", it.Description)
		if err != nil {
			return err
		}
		if v != "" {
			*description = v
		}
		v, err = promptLineDefault("Prompt (blank keeps current)", it.Prompt)
		if err != nil {
			return err
		}
		if v != "" {
			*prompt = v
		}
		curNotes := ""
		if it.Notes != nil {
			curNotes = *it.Notes
		}
		v, err = promptLineDefault("Notes (blank keeps current; type 'CLEAR' to remove)", curNotes)
		if err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(v), "CLEAR") {
			*clearNotes = true
		} else if v != "" {
			*notes = v
		}
	}

	changed := false
	if *title != "" {
		it.Title = *title
		changed = true
	}
	if *clearDescription {
		it.Description = ""
		changed = true
	} else if *description != "" {
		it.Description = *description
		changed = true
	}
	if *clearPrompt {
		it.Prompt = ""
		changed = true
	} else if *prompt != "" {
		it.Prompt = *prompt
		changed = true
	}

	if *clearNotes {
		it.Notes = nil
		changed = true
	} else if *notes != "" {
		n := *notes
		it.Notes = &n
		changed = true
	}

	if *acClear {
		it.AcceptanceCriteria = []string{}
		changed = true
	} else if len(ac) > 0 {
		it.AcceptanceCriteria = []string(ac)
		changed = true
	}

	if strings.TrimSpace(it.Title) == "" {
		return UsageError{Message: "title cannot be empty"}
	}

	if changed {
		it.UpdatedAt = time.Now().UTC()
		g.Items[id] = it
	}

	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "updated %s\n", id)
	return nil
}

func runDelete(id string, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cascade := fs.Bool("cascade-children", false, "delete this node and all descendants")
	force := fs.Bool("force", false, "also remove dep edges from remaining nodes that depend on deleted nodes")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "delete takes only flags (no positional args after <id>)"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	res, err := plan.DeleteItem(&g, id, *cascade, *force, now)
	if err != nil {
		return err
	}

	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "deleted %d item(s)\n", len(res.DeletedIDs))
	if len(res.DetachedIDs) > 0 {
		fmt.Fprintf(os.Stdout, "detached deps from: %s\n", strings.Join(res.DetachedIDs, ", "))
	}
	return nil
}

func runMove(id string, args []string) error {
	fs := flag.NewFlagSet("move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	parent := fs.String("parent", "", "parent id or 'root' (required)")
	indexStr := fs.String("index", "", "insert index within parent's childIds (optional)")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "move takes only flags (no positional args after <id>)"}
	}
	if strings.TrimSpace(*parent) == "" {
		return UsageError{Message: "move requires --parent <parentId|root>"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	var parentID *string
	if *parent != "root" {
		p := *parent
		parentID = &p
	}
	var idx *int
	if strings.TrimSpace(*indexStr) != "" {
		n, err := parseInt(*indexStr)
		if err != nil {
			return UsageError{Message: err.Error()}
		}
		idx = &n
	}

	now := time.Now().UTC()
	if err := plan.MoveItem(&g, id, parentID, idx, now); err != nil {
		return err
	}

	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "moved %s\n", id)
	return nil
}

func runDeps(args []string) error {
	if len(args) == 0 {
		return UsageError{Message: "deps requires a subcommand: add|remove|set|infer"}
	}
	switch args[0] {
	case "add":
		if len(args) != 3 {
			return UsageError{Message: "deps add requires: <id> <depId>"}
		}
		return runDepsAdd(args[1], args[2])
	case "remove":
		if len(args) != 3 {
			return UsageError{Message: "deps remove requires: <id> <depId>"}
		}
		return runDepsRemove(args[1], args[2])
	case "set":
		if len(args) < 2 {
			return UsageError{Message: "deps set requires: <id> [<depId> ...]"}
		}
		return runDepsSet(args[1], args[2:])
	case "infer":
		return runDepsInfer(args[1:])
	default:
		return UsageError{Message: fmt.Sprintf("unknown deps subcommand: %q", args[0])}
	}
}

func runDepsAdd(id, depID string) error {
	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := plan.AddDep(&g, id, depID, now); err != nil {
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "added dep %s -> %s\n", id, depID)
	return nil
}

func runDepsRemove(id, depID string) error {
	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := plan.RemoveDep(&g, id, depID, now); err != nil {
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "removed dep %s -> %s\n", id, depID)
	return nil
}

func runDepsSet(id string, deps []string) error {
	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := plan.SetDeps(&g, id, deps, now); err != nil {
		return err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		printValidationErrors(os.Stdout, path, errs)
		return errors.New("validation failed")
	}
	if err := plan.SaveAtomic(path, g); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "set %s deps (%d)\n", id, len(deps))
	return nil
}

func loadValidatedPlan(path string) (plan.WorkGraph, error) {
	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return plan.WorkGraph{}, fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return plan.WorkGraph{}, err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return plan.WorkGraph{}, fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path)
	}
	return g, nil
}

func printValidationErrors(w io.Writer, path string, errs []plan.ValidationError) {
	fmt.Fprintf(w, "invalid plan: %s\n", path)
	for _, e := range errs {
		fmt.Fprintf(w, "- %s: %s\n", e.Path, e.Message)
	}
}

func generateID(g plan.WorkGraph) string {
	// Small, stable-enough ID for human CLI usage: "w_<hex>".
	// (Generated only on creation; never re-generated.)
	for {
		var b [6]byte
		_, _ = rand.Read(b[:])
		id := "w_" + hex.EncodeToString(b[:])
		if g.Items == nil {
			return id
		}
		if _, ok := g.Items[id]; !ok {
			return id
		}
	}
}

func promptLine(label string) (string, error) {
	if label != "" {
		fmt.Fprintf(os.Stdout, "%s: ", label)
	}
	line, err := promptReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptLineDefault(label string, cur string) (string, error) {
	if cur != "" {
		fmt.Fprintf(os.Stdout, "%s [%s]: ", label, cur)
	} else {
		fmt.Fprintf(os.Stdout, "%s: ", label)
	}
	line, err := promptReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("index must be a number")
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return n, nil
}
