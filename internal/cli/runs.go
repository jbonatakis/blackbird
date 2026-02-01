package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runRuns(args []string) error {
	fs := flag.NewFlagSet("runs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	verbose := fs.Bool("verbose", false, "show full stdout/stderr")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 1 {
		return UsageError{Message: "runs requires exactly 1 argument: <taskID>"}
	}

	taskID := fs.Arg(0)
	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}
	if _, ok := g.Items[taskID]; !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}

	baseDir := filepath.Dir(path)
	records, err := execution.ListRuns(baseDir, taskID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Fprintf(os.Stdout, "no runs found for %s\n", taskID)
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "Run ID\tStarted\tDuration\tStatus\tExit Code")
	for _, record := range records {
		exitCode := "-"
		if record.ExitCode != nil {
			exitCode = fmt.Sprintf("%d", *record.ExitCode)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			record.ID,
			record.StartedAt.UTC().Format(time.RFC3339),
			formatRunDuration(record),
			record.Status,
			exitCode,
		)
	}
	_ = tw.Flush()

	if *verbose {
		for _, record := range records {
			fmt.Fprintf(os.Stdout, "\nRun %s\n", record.ID)
			fmt.Fprintln(os.Stdout, "Stdout:")
			if record.Stdout != "" {
				fmt.Fprintln(os.Stdout, record.Stdout)
			} else {
				fmt.Fprintln(os.Stdout, "(empty)")
			}
			fmt.Fprintln(os.Stdout, "Stderr:")
			if record.Stderr != "" {
				fmt.Fprintln(os.Stdout, record.Stderr)
			} else {
				fmt.Fprintln(os.Stdout, "(empty)")
			}
		}
	}

	return nil
}

func formatRunDuration(record execution.RunRecord) string {
	if record.CompletedAt == nil {
		return "running"
	}
	duration := record.CompletedAt.Sub(record.StartedAt)
	if duration < 0 {
		duration = 0
	}
	return duration.Truncate(time.Second).String()
}
