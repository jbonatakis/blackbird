package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
