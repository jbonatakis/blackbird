package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

const (
	maxAgentQuestionRounds = 2
	maxGenerateRevisions   = 1
)

func runPlan(args []string) error {
	if len(args) == 0 {
		return UsageError{Message: "plan requires a subcommand: generate|refine"}
	}
	switch args[0] {
	case "generate":
		return runPlanGenerate(args[1:])
	case "refine":
		return runPlanRefine(args[1:])
	default:
		return UsageError{Message: fmt.Sprintf("unknown plan subcommand: %q", args[0])}
	}
}

func runPlanGenerate(args []string) error {
	fs := flag.NewFlagSet("plan generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	description := fs.String("description", "", "project description")
	granularity := fs.String("granularity", "", "desired granularity (optional)")
	var constraints multiStringFlag
	fs.Var(&constraints, "constraint", "constraint (repeatable)")
	meta := addAgentMetaFlags(fs)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "plan generate takes only flags (no positional args)"}
	}

	if strings.TrimSpace(*description) == "" {
		v, err := promptLine("Project description")
		if err != nil {
			return err
		}
		*description = v
	}
	if strings.TrimSpace(*description) == "" {
		return UsageError{Message: "project description is required (use --description or enter it interactively)"}
	}

	if len(constraints) == 0 {
		v, err := promptLine("Constraints (optional; comma-separated)")
		if err != nil {
			return err
		}
		if strings.TrimSpace(v) != "" {
			constraints = splitCommaList(v)
		}
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	path := planPath()
	exists, existing, err := loadPlanIfExists(path)
	if err != nil {
		return err
	}
	if exists && len(existing.Items) > 0 {
		confirm, err := promptConfirm(fmt.Sprintf("Plan already exists with %d items. Overwrite", len(existing.Items)))
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Fprintln(os.Stdout, "aborted; plan unchanged")
			return nil
		}
	}

	requestMeta := buildAgentMetadata(meta)
	requestMeta.JSONSchema = defaultPlanJSONSchema()
	req := agent.Request{
		SchemaVersion:      agent.SchemaVersion,
		Type:               agent.RequestPlanGenerate,
		SystemPrompt:       defaultPlanSystemPrompt(),
		ProjectDescription: strings.TrimSpace(*description),
		Constraints:        trimNonEmpty(constraints),
		Granularity:        strings.TrimSpace(*granularity),
		Metadata:           requestMeta,
	}

	var proposed plan.WorkGraph
	diag := agent.Diagnostics{}
	resp, diag, err := runAgentWithQuestions(context.Background(), runtime, req, maxAgentQuestionRounds)
	if err != nil {
		return formatAgentRunError(err, diag)
	}
	next, err := responseToPlan(plan.NewEmptyWorkGraph(), resp, time.Now().UTC())
	if err != nil {
		return err
	}
	proposed = next

	revisions := 0
	for {
		fmt.Fprintln(os.Stdout, "")
		printProviderSummary(os.Stdout, runtime, req.Metadata)
		printPlanSummary(os.Stdout, proposed)
		fmt.Fprintln(os.Stdout, "Plan tree:")
		printTree(os.Stdout, proposed)

		choice, err := promptChoice("Accept plan", []string{"yes", "revise", "no"})
		if err != nil {
			return err
		}
		switch choice {
		case "yes":
			if err := plan.SaveAtomic(path, proposed); err != nil {
				return fmt.Errorf("write plan file: %w", err)
			}
			fmt.Fprintf(os.Stdout, "saved plan: %s\n", path)
			return nil
		case "revise":
			if revisions >= maxGenerateRevisions {
				return errors.New("revision limit reached")
			}
			revisions++
			change, err := promptLine("Revision request")
			if err != nil {
				return err
			}
			if strings.TrimSpace(change) == "" {
				return UsageError{Message: "revision request cannot be empty"}
			}
			revisionMeta := buildAgentMetadata(meta)
			revisionMeta.JSONSchema = defaultPlanJSONSchema()
			refineReq := agent.Request{
				SchemaVersion: agent.SchemaVersion,
				Type:          agent.RequestPlanRefine,
				SystemPrompt:  defaultPlanSystemPrompt(),
				ChangeRequest: strings.TrimSpace(change),
				Plan:          &proposed,
				Metadata:      revisionMeta,
			}
			resp, diag, err = runAgentWithQuestions(context.Background(), runtime, refineReq, maxAgentQuestionRounds)
			if err != nil {
				return formatAgentRunError(err, diag)
			}
			next, err := responseToPlan(proposed, resp, time.Now().UTC())
			if err != nil {
				return err
			}
			proposed = next
		default:
			fmt.Fprintln(os.Stdout, "aborted; plan unchanged")
			return nil
		}
	}
}

func runPlanRefine(args []string) error {
	fs := flag.NewFlagSet("plan refine", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	change := fs.String("change", "", "change request")
	meta := addAgentMetaFlags(fs)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "plan refine takes only flags (no positional args)"}
	}

	if strings.TrimSpace(*change) == "" {
		v, err := promptLine("Change request")
		if err != nil {
			return err
		}
		*change = v
	}
	if strings.TrimSpace(*change) == "" {
		return UsageError{Message: "change request is required (use --change or enter it interactively)"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	requestMeta := buildAgentMetadata(meta)
	requestMeta.JSONSchema = defaultPlanJSONSchema()
	req := agent.Request{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		SystemPrompt:  defaultPlanSystemPrompt(),
		ChangeRequest: strings.TrimSpace(*change),
		Plan:          &g,
		Metadata:      requestMeta,
	}

	resp, diag, err := runAgentWithQuestions(context.Background(), runtime, req, maxAgentQuestionRounds)
	if err != nil {
		return formatAgentRunError(err, diag)
	}

	next, err := responseToPlan(g, resp, time.Now().UTC())
	if err != nil {
		return err
	}

	diff := plan.Diff(g, next)
	fmt.Fprintln(os.Stdout, "")
	printProviderSummary(os.Stdout, runtime, req.Metadata)
	printDiffSummary(os.Stdout, diff)

	if err := plan.SaveAtomic(path, next); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "updated plan: %s\n", path)
	return nil
}

func runDepsInfer(args []string) error {
	fs := flag.NewFlagSet("deps infer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var hints multiStringFlag
	fs.Var(&hints, "hint", "dependency hint (repeatable)")
	meta := addAgentMetaFlags(fs)

	if err := fs.Parse(args); err != nil {
		return UsageError{Message: err.Error()}
	}
	if fs.NArg() != 0 {
		return UsageError{Message: "deps infer takes only flags (no positional args)"}
	}

	path := planPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	requestMeta := buildAgentMetadata(meta)
	requestMeta.JSONSchema = defaultPlanJSONSchema()
	req := agent.Request{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestDepsInfer,
		SystemPrompt:  defaultPlanSystemPrompt(),
		Plan:          &g,
		Constraints:   trimNonEmpty(hints),
		Metadata:      requestMeta,
	}

	resp, diag, err := runAgentWithQuestions(context.Background(), runtime, req, maxAgentQuestionRounds)
	if err != nil {
		return formatAgentRunError(err, diag)
	}

	next, err := responseToPlan(g, resp, time.Now().UTC())
	if err != nil {
		return err
	}

	diff := plan.Diff(g, next)
	fmt.Fprintln(os.Stdout, "")
	printProviderSummary(os.Stdout, runtime, req.Metadata)
	printDiffSummary(os.Stdout, diff)
	printDepsRationaleExcerpt(os.Stdout, next, 6)

	confirm, err := promptConfirm("Apply dependency updates")
	if err != nil {
		return err
	}
	if !confirm {
		fmt.Fprintln(os.Stdout, "aborted; plan unchanged")
		return nil
	}

	if err := plan.SaveAtomic(path, next); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "updated plan: %s\n", path)
	return nil
}
