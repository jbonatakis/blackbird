package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
	"github.com/jbonatakis/blackbird/internal/memory/contextpack"
	"github.com/jbonatakis/blackbird/internal/memory/index"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func runMem(args []string) error {
	if len(args) == 0 {
		return UsageError{Message: "mem requires a subcommand: search|get|context"}
	}

	switch args[0] {
	case "search":
		return runMemSearch(args[1:])
	case "get":
		return runMemGet(args[1:])
	case "context":
		return runMemContext(args[1:])
	default:
		return UsageError{Message: fmt.Sprintf("unknown mem subcommand: %q", args[0])}
	}
}

func runMemSearch(args []string) error {
	fs := flag.NewFlagSet("mem search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	sessionID := fs.String("session", "", "filter by session id")
	taskID := fs.String("task", "", "filter by task id")
	runID := fs.String("run", "", "filter by run id")
	types := fs.String("type", "", "comma-separated artifact types")
	limit := fs.Int("limit", 0, "limit results")
	offset := fs.Int("offset", 0, "offset results")
	snippetMax := fs.Int("snippet-max", 0, "max snippet length")
	snippetTokens := fs.Int("snippet-tokens", 0, "max snippet tokens")

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if *limit < 0 {
		return UsageError{Message: "limit must be >= 0"}
	}
	if *offset < 0 {
		return UsageError{Message: "offset must be >= 0"}
	}
	if *snippetMax < 0 {
		return UsageError{Message: "snippet-max must be >= 0"}
	}
	if *snippetTokens < 0 {
		return UsageError{Message: "snippet-tokens must be >= 0"}
	}
	if fs.NArg() == 0 {
		return UsageError{Message: "mem search requires a query"}
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return UsageError{Message: "mem search requires a query"}
	}

	parsedTypes, err := parseArtifactTypes(*types)
	if err != nil {
		return UsageError{Message: err.Error()}
	}

	baseDir := filepath.Dir(plan.PlanPath())
	opts := index.SearchOptions{
		Query:         query,
		SessionID:     strings.TrimSpace(*sessionID),
		TaskID:        strings.TrimSpace(*taskID),
		RunID:         strings.TrimSpace(*runID),
		Types:         parsedTypes,
		Limit:         *limit,
		Offset:        *offset,
		SnippetMaxLen: *snippetMax,
		SnippetTokens: *snippetTokens,
	}
	cards, err := index.SearchForProject(baseDir, opts)
	if err != nil {
		return err
	}
	if len(cards) == 0 {
		fmt.Fprintln(os.Stdout, "no results")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "Artifact ID\tType\tSession\tTask\tRun\tCreated\tSnippet")
	for _, card := range cards {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			card.ArtifactID,
			card.Type,
			formatOptional(card.SessionID),
			formatOptional(card.TaskID),
			formatOptional(card.RunID),
			formatOptional(formatTime(card.CreatedAt)),
			formatOptional(formatSnippet(card.Snippet)),
		)
	}
	_ = tw.Flush()
	return nil
}

func runMemGet(args []string) error {
	fs := flag.NewFlagSet("mem get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 1 {
		return UsageError{Message: "mem get requires exactly 1 argument: <artifact_id>"}
	}

	artifactID := strings.TrimSpace(fs.Arg(0))
	if artifactID == "" {
		return UsageError{Message: "artifact id is required"}
	}

	baseDir := filepath.Dir(plan.PlanPath())
	art, ok, err := index.GetForProject(baseDir, artifactID)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintf(os.Stdout, "artifact not found: %s\n", artifactID)
		return nil
	}

	payload, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return fmt.Errorf("encode artifact: %w", err)
	}
	payload = append(payload, '\n')
	if _, err := os.Stdout.Write(payload); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}
	return nil
}

func runMemContext(args []string) error {
	fs := flag.NewFlagSet("mem context", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	taskIDFlag := fs.String("task", "", "task id to build context pack for")
	sessionOverride := fs.String("session", "", "override session id")
	goalOverride := fs.String("goal", "", "override session goal")
	budgetFlag := intOverrideFlag{}
	fs.Func("budget", "total token budget override", budgetFlag.Set)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}

	taskID := strings.TrimSpace(*taskIDFlag)
	if taskID == "" {
		switch fs.NArg() {
		case 0:
			return UsageError{Message: "mem context requires a task id (--task <id>)"}
		case 1:
			taskID = strings.TrimSpace(fs.Arg(0))
		default:
			return UsageError{Message: "mem context requires exactly 1 task id"}
		}
	} else if fs.NArg() != 0 {
		return UsageError{Message: "mem context task id provided twice"}
	}
	if taskID == "" {
		return UsageError{Message: "mem context requires a task id (--task <id>)"}
	}

	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}
	if _, ok := g.Items[taskID]; !ok {
		return fmt.Errorf("unknown id %q", taskID)
	}

	baseDir := filepath.Dir(path)
	configResolved, err := config.LoadConfig(baseDir)
	if err != nil {
		return err
	}

	sessionID := strings.TrimSpace(*sessionOverride)
	sessionGoal := strings.TrimSpace(*goalOverride)
	if sessionID == "" || sessionGoal == "" {
		session, present, err := memory.LoadSession(memory.SessionPath(baseDir))
		if err != nil {
			return err
		}
		if present {
			if sessionID == "" {
				sessionID = session.SessionID
			}
			if sessionGoal == "" {
				sessionGoal = session.Goal
			}
		}
	}
	if sessionID == "" {
		return fmt.Errorf("session metadata not found (expected %s); use --session to override", memory.SessionPath(baseDir))
	}

	store, _, err := artifact.LoadStoreForProject(baseDir)
	if err != nil {
		return err
	}

	budget := budgetFromConfig(configResolved)
	if budgetFlag.set {
		budget.TotalTokens = budgetFlag.value
	}

	pack := contextpack.Build(contextpack.BuildOptions{
		SessionID:     sessionID,
		SessionGoal:   sessionGoal,
		Artifacts:     store.Artifacts,
		Budget:        budget,
		RunTimeLookup: execution.RunTimeLookupFromExecution(baseDir),
	})

	fmt.Fprintln(os.Stdout, contextpack.Render(pack))
	return nil
}

type intOverrideFlag struct {
	set   bool
	value int
}

func (f *intOverrideFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid budget %q", value)
	}
	if n < 0 {
		return fmt.Errorf("budget must be >= 0")
	}
	f.value = n
	f.set = true
	return nil
}

func budgetFromConfig(cfg config.ResolvedConfig) contextpack.Budget {
	budgets := cfg.Memory.Budgets
	return contextpack.Budget{
		TotalTokens:            budgets.TotalTokens,
		DecisionsTokens:        budgets.DecisionsTokens,
		ConstraintsTokens:      budgets.ConstraintsTokens,
		ImplementedTokens:      budgets.ImplementedTokens,
		OpenThreadsTokens:      budgets.OpenThreadsTokens,
		ArtifactPointersTokens: budgets.ArtifactPointersTokens,
	}
}

func parseArtifactTypes(raw string) ([]artifact.ArtifactType, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]artifact.ArtifactType, 0, len(parts))
	for _, part := range parts {
		typeName := strings.TrimSpace(part)
		if typeName == "" {
			continue
		}
		typ := artifact.ArtifactType(typeName)
		switch typ {
		case artifact.ArtifactOutcome, artifact.ArtifactDecision, artifact.ArtifactConstraint, artifact.ArtifactOpenThread, artifact.ArtifactTranscript:
			out = append(out, typ)
		default:
			return nil, fmt.Errorf("unknown artifact type %q", typeName)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func formatOptional(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func formatSnippet(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	return value
}
