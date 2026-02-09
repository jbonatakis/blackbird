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
	"github.com/jbonatakis/blackbird/internal/plangen"
	"github.com/jbonatakis/blackbird/internal/planquality"
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

	path := plan.PlanPath()
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
	requestMeta.JSONSchema = agent.DefaultPlanJSONSchema()
	requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)
	runRequest := func(ctx context.Context, req agent.Request) (agent.Response, agent.Diagnostics, error) {
		return runAgentWithQuestions(ctx, runtime, req, agent.MaxPlanQuestionRounds)
	}
	qualityGatePassLimit := plangen.ResolveMaxAutoRefinePassesFromPlanPath(path)
	refineViaPlanRequest := func(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) (plan.WorkGraph, error) {
		refineReqMeta := buildAgentMetadata(meta)
		refineReqMeta.JSONSchema = agent.DefaultPlanJSONSchema()
		refineReqMeta = agent.ApplyRuntimeProvider(refineReqMeta, runtime)

		refineResult, err := plangen.Refine(ctx, runRequest, plangen.RefineInput{
			ChangeRequest: strings.TrimSpace(changeRequest),
			CurrentPlan:   currentPlan,
			Metadata:      refineReqMeta,
		})
		if err != nil {
			return plan.WorkGraph{}, formatAgentRunError(err, refineResult.Diagnostics)
		}
		if len(refineResult.Questions) > 0 {
			return plan.WorkGraph{}, errors.New("unexpected clarification request during quality auto-refine")
		}
		if refineResult.Plan == nil {
			return plan.WorkGraph{}, errors.New("plan refine returned no plan")
		}
		return plan.Clone(*refineResult.Plan), nil
	}
	applyQualityGate := func(candidate plan.WorkGraph) (planquality.QualityGateResult, error) {
		initialFindings := planquality.Lint(candidate)
		printQualitySummary(os.Stdout, "initial", initialFindings)

		autoRefinePass := 0
		result, err := plangen.RunQualityGate(context.Background(), candidate, qualityGatePassLimit, func(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) (plan.WorkGraph, error) {
			autoRefinePass++
			fmt.Fprintf(os.Stdout, "quality auto-refine pass %d/%d\n", autoRefinePass, qualityGatePassLimit)
			return refineViaPlanRequest(ctx, changeRequest, currentPlan)
		})
		if err != nil {
			return planquality.QualityGateResult{}, err
		}

		finalSummary := printQualitySummary(os.Stdout, "final", result.FinalFindings)
		if finalSummary.Blocking > 0 {
			printQualityFindings(os.Stdout, "Blocking findings:", result.FinalFindings, planquality.SeverityBlocking)
		}
		if finalSummary.Warning > 0 {
			printQualityFindings(os.Stdout, "Warning findings (non-blocking):", result.FinalFindings, planquality.SeverityWarning)
		}

		return result, nil
	}

	var proposed plan.WorkGraph
	generateResult, err := plangen.Generate(context.Background(), runRequest, plangen.GenerateInput{
		Description: strings.TrimSpace(*description),
		Constraints: trimNonEmpty(constraints),
		Granularity: strings.TrimSpace(*granularity),
		Metadata:    requestMeta,
	})
	if err != nil {
		return formatAgentRunError(err, generateResult.Diagnostics)
	}
	if len(generateResult.Questions) > 0 {
		return errors.New("unexpected clarification request during plan generation")
	}
	if generateResult.Plan == nil {
		return errors.New("plan generation returned no plan")
	}
	proposed = plan.Clone(*generateResult.Plan)

	qualityResult, err := applyQualityGate(proposed)
	if err != nil {
		return err
	}
	proposed = plan.Clone(qualityResult.FinalPlan)

	revisions := 0
	applyManualRevision := func(current plan.WorkGraph) (plan.WorkGraph, planquality.QualityGateResult, error) {
		if revisions >= agent.MaxPlanGenerateRevisions {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, errors.New("revision limit reached")
		}
		revisions++
		change, err := promptLine("Revision request")
		if err != nil {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, err
		}
		if strings.TrimSpace(change) == "" {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, UsageError{Message: "revision request cannot be empty"}
		}
		revisionMeta := buildAgentMetadata(meta)
		revisionMeta.JSONSchema = agent.DefaultPlanJSONSchema()
		revisionMeta = agent.ApplyRuntimeProvider(revisionMeta, runtime)
		refineResult, err := plangen.Refine(context.Background(), runRequest, plangen.RefineInput{
			ChangeRequest: strings.TrimSpace(change),
			CurrentPlan:   current,
			Metadata:      revisionMeta,
		})
		if err != nil {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, formatAgentRunError(err, refineResult.Diagnostics)
		}
		if len(refineResult.Questions) > 0 {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, errors.New("unexpected clarification request during plan revision")
		}
		if refineResult.Plan == nil {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, errors.New("plan revision returned no plan")
		}
		next := plan.Clone(*refineResult.Plan)
		result, err := applyQualityGate(next)
		if err != nil {
			return plan.WorkGraph{}, planquality.QualityGateResult{}, err
		}
		return plan.Clone(result.FinalPlan), result, nil
	}

	for {
		fmt.Fprintln(os.Stdout, "")
		printProviderSummary(os.Stdout, runtime, requestMeta)
		printPlanSummary(os.Stdout, proposed)
		fmt.Fprintln(os.Stdout, "Plan tree:")
		printTree(os.Stdout, proposed)

		if planquality.HasBlocking(qualityResult.FinalFindings) {
			choice, err := promptChoice("Blocking findings remain. Choose action", []string{"revise", "accept_anyway", "cancel"})
			if err != nil {
				return err
			}
			switch choice {
			case "revise":
				proposed, qualityResult, err = applyManualRevision(proposed)
				if err != nil {
					return err
				}
			case "accept_anyway":
				fmt.Fprintln(os.Stdout, "WARNING: blocking findings were overridden; saving plan anyway")
				if err := plan.SaveAtomic(path, proposed); err != nil {
					return fmt.Errorf("write plan file: %w", err)
				}
				fmt.Fprintf(os.Stdout, "saved plan: %s\n", path)
				return nil
			default:
				fmt.Fprintln(os.Stdout, "aborted; plan unchanged")
				return nil
			}
			continue
		}

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
			proposed, qualityResult, err = applyManualRevision(proposed)
			if err != nil {
				return err
			}
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

	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	requestMeta := buildAgentMetadata(meta)
	requestMeta.JSONSchema = agent.DefaultPlanJSONSchema()
	requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)
	req := agent.Request{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		SystemPrompt:  agent.DefaultPlanSystemPrompt(),
		ChangeRequest: strings.TrimSpace(*change),
		Plan:          &g,
		Metadata:      requestMeta,
	}

	resp, diag, err := runAgentWithQuestions(context.Background(), runtime, req, agent.MaxPlanQuestionRounds)
	if err != nil {
		return formatAgentRunError(err, diag)
	}

	next, err := agent.ResponseToPlan(g, resp, time.Now().UTC())
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

	path := plan.PlanPath()
	g, err := loadValidatedPlan(path)
	if err != nil {
		return err
	}

	runtime, err := agent.NewRuntimeFromEnv()
	if err != nil {
		return err
	}

	requestMeta := buildAgentMetadata(meta)
	requestMeta.JSONSchema = agent.DefaultPlanJSONSchema()
	requestMeta = agent.ApplyRuntimeProvider(requestMeta, runtime)
	req := agent.Request{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestDepsInfer,
		SystemPrompt:  agent.DefaultPlanSystemPrompt(),
		Plan:          &g,
		Constraints:   trimNonEmpty(hints),
		Metadata:      requestMeta,
	}

	resp, diag, err := runAgentWithQuestions(context.Background(), runtime, req, agent.MaxPlanQuestionRounds)
	if err != nil {
		return formatAgentRunError(err, diag)
	}

	next, err := agent.ResponseToPlan(g, resp, time.Now().UTC())
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
