package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type agentMetaFlags struct {
	model          string
	maxTokens      int
	maxTokensSet   bool
	temperature    float64
	temperatureSet bool
	responseFormat string
}

type rationaleEntry struct {
	key   string
	value string
}

func addAgentMetaFlags(fs *flag.FlagSet) *agentMetaFlags {
	meta := &agentMetaFlags{}
	fs.StringVar(&meta.model, "model", "", "provider model name")
	fs.StringVar(&meta.responseFormat, "response-format", "", "response format (if provider supports)")
	fs.Func("max-tokens", "max tokens for response", func(v string) error {
		if v == "" {
			return nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid max-tokens %q", v)
		}
		meta.maxTokens = n
		meta.maxTokensSet = true
		return nil
	})
	fs.Func("temperature", "sampling temperature", func(v string) error {
		if v == "" {
			return nil
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid temperature %q", v)
		}
		meta.temperature = f
		meta.temperatureSet = true
		return nil
	})
	return meta
}

func buildAgentMetadata(meta *agentMetaFlags) agent.RequestMetadata {
	out := agent.RequestMetadata{
		Model:          strings.TrimSpace(meta.model),
		ResponseFormat: strings.TrimSpace(meta.responseFormat),
	}
	if meta.maxTokensSet {
		out.MaxTokens = &meta.maxTokens
	}
	if meta.temperatureSet {
		out.Temperature = &meta.temperature
	}
	return out
}

func defaultPlanJSONSchema() string {
	return strings.TrimSpace(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["schemaVersion", "type"],
  "properties": {
    "schemaVersion": { "type": "integer" },
    "type": { "type": "string", "enum": ["plan_generate", "plan_refine", "deps_infer"] },
    "plan": { "$ref": "#/definitions/workGraph" },
    "patch": { "type": "array", "items": { "$ref": "#/definitions/patchOp" } },
    "questions": { "type": "array", "items": { "$ref": "#/definitions/question" } }
  },
  "oneOf": [
    { "required": ["plan"] },
    { "required": ["patch"] },
    { "required": ["questions"] }
  ],
  "definitions": {
    "workGraph": {
      "type": "object",
      "required": ["schemaVersion", "items"],
      "properties": {
        "schemaVersion": { "type": "integer" },
        "items": {
          "type": "object",
          "additionalProperties": { "$ref": "#/definitions/workItem" }
        }
      }
    },
    "workItem": {
      "type": "object",
      "required": [
        "id", "title", "description", "acceptanceCriteria", "prompt",
        "parentId", "childIds", "deps", "status", "createdAt", "updatedAt"
      ],
      "properties": {
        "id": { "type": "string" },
        "title": { "type": "string" },
        "description": { "type": "string" },
        "acceptanceCriteria": { "type": "array", "items": { "type": "string" } },
        "prompt": { "type": "string" },
        "parentId": { "type": ["string", "null"] },
        "childIds": { "type": "array", "items": { "type": "string" } },
        "deps": { "type": "array", "items": { "type": "string" } },
        "status": { "type": "string", "enum": ["todo", "in_progress", "blocked", "done", "skipped"] },
        "createdAt": { "type": "string", "format": "date-time" },
        "updatedAt": { "type": "string", "format": "date-time" },
        "notes": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "patchOp": {
      "type": "object",
      "required": ["op"],
      "properties": {
        "op": { "type": "string", "enum": ["add", "update", "delete", "move", "set_deps", "add_dep", "remove_dep"] },
        "id": { "type": "string" },
        "item": { "$ref": "#/definitions/workItem" },
        "parentId": { "type": ["string", "null"] },
        "index": { "type": "integer", "minimum": 0 },
        "deps": { "type": "array", "items": { "type": "string" } },
        "depId": { "type": "string" },
        "rationale": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "question": {
      "type": "object",
      "required": ["id", "prompt"],
      "properties": {
        "id": { "type": "string" },
        "prompt": { "type": "string" },
        "options": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}`)
}

func defaultPlanSystemPrompt() string {
	return strings.TrimSpace("You are a planning agent for blackbird.\n\n" +
		"Return exactly one JSON object on stdout (or a single fenced ```json block).\n" +
		"Do not include any other text outside the JSON.\n\n" +
		"Response shape:\n" +
		"- Must include schemaVersion and type.\n" +
		"- Must include exactly one of: plan, patch, or questions.\n\n" +
		"Plan requirements:\n" +
		"- Plan must conform to the WorkGraph schema.\n" +
		"- Every WorkItem must include required fields: id, title, description, acceptanceCriteria, prompt, parentId, childIds, deps, status, createdAt, updatedAt.\n" +
		"- Use stable, unique ids and keep parent/child relationships consistent.\n" +
		"- Deps must reference existing ids and must not form cycles.\n\n" +
		"- Avoid meta tasks like \"design the app\" or \"plan the work\" unless explicitly requested; the plan itself is the design.\n" +
		"- Top-level features should be meaningful deliverables, not a generic \"root\" placeholder.\n\n" +
		"Patch requirements:\n" +
		"- Use only ops: add, update, delete, move, set_deps, add_dep, remove_dep.\n" +
		"- Include required fields for each op.\n" +
		"- Do not introduce cycles or invalid references.\n\n" +
		"Questions:\n" +
		"- If clarification is required, respond with questions only (no plan/patch).\n" +
		"- Each question must include id and prompt; options are optional.\n")
}

func runAgentWithQuestions(ctx context.Context, runtime agent.Runtime, req agent.Request, maxRounds int) (agent.Response, agent.Diagnostics, error) {
	if maxRounds < 0 {
		maxRounds = 0
	}

	cur := req
	for round := 0; round <= maxRounds; round++ {
		stopProgress := startProgressIndicator(os.Stderr, "Running agent", time.Second)
		resp, diag, err := runtime.Run(ctx, cur)
		stopProgress()
		if err != nil {
			return agent.Response{}, diag, err
		}
		if len(resp.Questions) == 0 {
			return resp, diag, nil
		}
		answers, err := promptAnswers(resp.Questions)
		if err != nil {
			return agent.Response{}, diag, err
		}
		cur.Answers = answers
	}
	return agent.Response{}, agent.Diagnostics{}, errors.New("too many clarification rounds")
}

func startProgressIndicator(w io.Writer, label string, interval time.Duration) func() {
	if interval <= 0 {
		interval = time.Second
	}
	fmt.Fprintf(w, "%s ", label)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprint(w, ".")
			}
		}
	}()
	return func() {
		close(done)
		wg.Wait()
		fmt.Fprintln(w, " done")
	}
}

func responseToPlan(base plan.WorkGraph, resp agent.Response, now time.Time) (plan.WorkGraph, error) {
	if resp.Plan != nil {
		return plan.NormalizeWorkGraphTimestamps(*resp.Plan, now), nil
	}
	if len(resp.Patch) == 0 {
		return plan.WorkGraph{}, errors.New("agent response contained no plan or patch")
	}
	next := plan.Clone(base)
	if err := agent.ApplyPatch(&next, resp.Patch, now); err != nil {
		return plan.WorkGraph{}, err
	}
	return next, nil
}

func formatAgentRunError(err error, diag agent.Diagnostics) error {
	if err == nil {
		return nil
	}
	if diag.Stderr == "" && diag.Stdout == "" {
		return err
	}
	var lines []string
	lines = append(lines, err.Error())
	if strings.TrimSpace(diag.Stderr) != "" {
		lines = append(lines, "agent stderr:")
		lines = append(lines, indentLines(diag.Stderr, "  ")...)
	}
	if strings.TrimSpace(diag.Stdout) != "" {
		lines = append(lines, "agent stdout:")
		lines = append(lines, indentLines(diag.Stdout, "  ")...)
	}
	return errors.New(strings.Join(lines, "\n"))
}

func indentLines(s, prefix string) []string {
	raw := strings.Split(strings.TrimRight(s, "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, prefix+line)
	}
	return out
}

func printProviderSummary(w io.Writer, runtime agent.Runtime, meta agent.RequestMetadata) {
	provider := runtime.Provider
	if strings.TrimSpace(provider) == "" {
		provider = "unknown"
	}
	label := fmt.Sprintf("Provider: %s", provider)
	if strings.TrimSpace(meta.Model) != "" {
		label += fmt.Sprintf(" (model: %s)", meta.Model)
	}
	fmt.Fprintln(w, label)
}

func printPlanSummary(w io.Writer, g plan.WorkGraph) {
	total := len(g.Items)
	rootIDs := make([]string, 0)
	for id, it := range g.Items {
		if it.ParentID == nil || *it.ParentID == "" {
			rootIDs = append(rootIDs, id)
		}
	}
	sort.Strings(rootIDs)
	fmt.Fprintf(w, "Plan summary: %d item(s), %d top-level feature(s)\n", total, len(rootIDs))
	if len(rootIDs) == 0 {
		return
	}
	fmt.Fprintln(w, "Top-level features:")
	for _, id := range rootIDs {
		it := g.Items[id]
		fmt.Fprintf(w, "- %s: %s\n", id, it.Title)
	}
}

func printDiffSummary(w io.Writer, diff plan.DiffSummary) {
	fmt.Fprintf(w, "Diff summary: added %d, removed %d, updated %d, moved %d, deps +%d/-%d\n",
		len(diff.Added), len(diff.Removed), len(diff.Updated), len(diff.Moved), len(diff.DepsAdded), len(diff.DepsRemoved))
	printIDs := func(label string, ids []string) {
		if len(ids) == 0 {
			return
		}
		fmt.Fprintf(w, "%s: %s\n", label, strings.Join(ids, ", "))
	}
	printIDs("Added", diff.Added)
	printIDs("Removed", diff.Removed)
	printIDs("Updated", diff.Updated)
	printIDs("Moved", diff.Moved)
	if len(diff.DepsAdded) > 0 {
		fmt.Fprintf(w, "Deps added: %s\n", formatEdges(diff.DepsAdded))
	}
	if len(diff.DepsRemoved) > 0 {
		fmt.Fprintf(w, "Deps removed: %s\n", formatEdges(diff.DepsRemoved))
	}
}

func printDepsRationaleExcerpt(w io.Writer, g plan.WorkGraph, max int) {
	excerpts := depRationaleExcerpt(g, max)
	if len(excerpts) == 0 {
		return
	}
	fmt.Fprintln(w, "Dependency rationale (excerpt):")
	for _, line := range excerpts {
		fmt.Fprintf(w, "- %s\n", line)
	}
}

func depRationaleExcerpt(g plan.WorkGraph, max int) []string {
	var entries []rationaleEntry
	for id, it := range g.Items {
		for depID, reason := range it.DepRationale {
			key := id + "->" + depID
			entries = append(entries, rationaleEntry{key: key, value: reason})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})
	if max <= 0 || len(entries) <= max {
		return formatRationale(entries)
	}
	return formatRationale(entries[:max])
}

func formatRationale(entries []rationaleEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, fmt.Sprintf("%s: %s", e.key, e.value))
	}
	return out
}

func formatEdges(edges []plan.DepEdge) string {
	parts := make([]string, 0, len(edges))
	for _, e := range edges {
		parts = append(parts, fmt.Sprintf("%s->%s", e.From, e.To))
	}
	return strings.Join(parts, ", ")
}

func promptConfirm(label string) (bool, error) {
	line, err := promptLine(label + " [y/N]")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func promptChoice(label string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", errors.New("no choices provided")
	}
	choiceMap := map[string]string{}
	for _, c := range choices {
		choiceMap[strings.ToLower(c)] = c
	}

	display := make([]string, 0, len(choices))
	for _, c := range choices {
		display = append(display, c)
	}

	for {
		line, err := promptLine(fmt.Sprintf("%s [%s]", label, strings.Join(display, "/")))
		if err != nil {
			return "", err
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer == "" {
			return choices[0], nil
		}
		if answer == "y" {
			if v, ok := choiceMap["yes"]; ok {
				return v, nil
			}
		}
		if answer == "n" {
			if v, ok := choiceMap["no"]; ok {
				return v, nil
			}
		}
		if v, ok := choiceMap[answer]; ok {
			return v, nil
		}
		fmt.Fprintln(os.Stdout, "Invalid choice. Please enter one of:", strings.Join(display, "/"))
	}
}

func promptAnswers(questions []agent.Question) ([]agent.Answer, error) {
	var answers []agent.Answer
	for _, q := range questions {
		if len(q.Options) > 0 {
			fmt.Fprintf(os.Stdout, "Question: %s\n", q.Prompt)
			for i, opt := range q.Options {
				fmt.Fprintf(os.Stdout, "  %d) %s\n", i+1, opt)
			}
			line, err := promptLine("Answer")
			if err != nil {
				return nil, err
			}
			val := strings.TrimSpace(line)
			if val == "" {
				return nil, fmt.Errorf("answer required for question %q", q.ID)
			}
			if idx, err := strconv.Atoi(val); err == nil {
				if idx < 1 || idx > len(q.Options) {
					return nil, fmt.Errorf("invalid option %d for question %q", idx, q.ID)
				}
				val = q.Options[idx-1]
			}
			answers = append(answers, agent.Answer{ID: q.ID, Value: val})
			continue
		}
		line, err := promptLine(q.Prompt)
		if err != nil {
			return nil, err
		}
		val := strings.TrimSpace(line)
		if val == "" {
			return nil, fmt.Errorf("answer required for question %q", q.ID)
		}
		answers = append(answers, agent.Answer{ID: q.ID, Value: val})
	}
	return answers, nil
}

func splitCommaList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func loadPlanIfExists(path string) (bool, plan.WorkGraph, error) {
	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return false, plan.WorkGraph{}, nil
		}
		return false, plan.WorkGraph{}, err
	}
	if errs := plan.Validate(g); len(errs) != 0 {
		return true, plan.WorkGraph{}, fmt.Errorf("plan is invalid (run `blackbird validate`): %s", path)
	}
	return true, g, nil
}
