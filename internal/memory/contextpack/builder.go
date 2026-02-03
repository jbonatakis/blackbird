package contextpack

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

var defaultInstructions = []string{
	"Use this context pack as the authoritative session state.",
	"Only call memory tools if you need missing details.",
	"Prefer memory.get(id) for referenced artifacts before broad searches.",
	"Limit tool calls: at most 1 memory.search and 2 memory.get.",
}

// DefaultInstructions returns a copy of the default instruction text.
func DefaultInstructions() []string {
	return append([]string(nil), defaultInstructions...)
}

// Build assembles a session context pack from artifacts.
func Build(opts BuildOptions) ContextPack {
	budget := normalizeBudget(opts.Budget)
	instructions := opts.Instructions
	if len(instructions) == 0 {
		instructions = DefaultInstructions()
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	entries := filterArtifacts(opts.Artifacts, strings.TrimSpace(opts.SessionID))

	decisionEntries := selectDecisions(entries, opts.RunTimeLookup, now)
	constraintEntries := selectConstraints(entries, opts.RunTimeLookup, now)
	outcomeEntries := selectOutcomes(entries, opts.RunTimeLookup, now)
	openThreadEntries := selectOpenThreads(entries, opts.RunTimeLookup, now)

	pack := ContextPack{
		SchemaVersion: SchemaVersion,
		SessionID:     strings.TrimSpace(opts.SessionID),
		SessionGoal:   strings.TrimSpace(opts.SessionGoal),
		Instructions:  append([]string(nil), instructions...),
		Budget:        budget,
		GeneratedAt:   now,
	}

	usage := Usage{}
	usage.GoalTokens = estimateTokens(goalLine(pack.SessionGoal))
	usage.InstructionTokens = estimateTokens(strings.Join(instructions, " "))

	var totalRemaining *int
	if budget.TotalTokens > 0 {
		remaining := budget.TotalTokens - usage.GoalTokens - usage.InstructionTokens
		if remaining < 0 {
			remaining = 0
		}
		totalRemaining = &remaining
	}

	pack.Decisions, usage.DecisionsTokens = buildSection(decisionEntries, budget.DecisionsTokens, totalRemaining, formatDecision)
	pack.Constraints, usage.ConstraintsTokens = buildSection(constraintEntries, budget.ConstraintsTokens, totalRemaining, formatConstraint)
	pack.Implemented, usage.ImplementedTokens = buildSection(outcomeEntries, budget.ImplementedTokens, totalRemaining, formatOutcome)
	pack.OpenThreads, usage.OpenThreadsTokens = buildSection(openThreadEntries, budget.OpenThreadsTokens, totalRemaining, formatOpenThread)

	artifactEntries := combineEntries(decisionEntries, constraintEntries, outcomeEntries, openThreadEntries)
	pack.ArtifactIDs, usage.ArtifactPointersTokens = buildSection(artifactEntries, budget.ArtifactPointersTokens, totalRemaining, formatArtifactPointer)

	usage.TotalTokens = usage.GoalTokens + usage.InstructionTokens + usage.DecisionsTokens + usage.ConstraintsTokens + usage.ImplementedTokens + usage.OpenThreadsTokens + usage.ArtifactPointersTokens
	pack.Usage = usage

	return pack
}

type artifactEntry struct {
	art artifact.Artifact
	ts  time.Time
	key string
}

func normalizeBudget(b Budget) Budget {
	b.TotalTokens = clampNonNegative(b.TotalTokens)
	b.DecisionsTokens = clampNonNegative(b.DecisionsTokens)
	b.ConstraintsTokens = clampNonNegative(b.ConstraintsTokens)
	b.ImplementedTokens = clampNonNegative(b.ImplementedTokens)
	b.OpenThreadsTokens = clampNonNegative(b.OpenThreadsTokens)
	b.ArtifactPointersTokens = clampNonNegative(b.ArtifactPointersTokens)
	return b
}

func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func filterArtifacts(artifacts []artifact.Artifact, sessionID string) []artifact.Artifact {
	if sessionID == "" {
		return append([]artifact.Artifact(nil), artifacts...)
	}
	filtered := make([]artifact.Artifact, 0, len(artifacts))
	for _, art := range artifacts {
		if strings.TrimSpace(art.SessionID) != sessionID {
			continue
		}
		filtered = append(filtered, art)
	}
	return filtered
}

func selectDecisions(artifacts []artifact.Artifact, lookup RunTimeLookup, fallback time.Time) []artifactEntry {
	entries := make([]artifactEntry, 0, len(artifacts))
	for _, art := range artifacts {
		if art.Type != artifact.ArtifactDecision {
			continue
		}
		if len(art.SupersededBy) > 0 {
			continue
		}
		key := normalizeStatement(art.Content.Text)
		if key == "" {
			continue
		}
		entries = append(entries, artifactEntry{art: art, ts: resolveTime(art, lookup, fallback), key: key})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].ts.Equal(entries[j].ts) {
			return entries[i].ts.After(entries[j].ts)
		}
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return entries[i].art.ArtifactID < entries[j].art.ArtifactID
	})

	seen := make(map[string]struct{}, len(entries))
	out := make([]artifactEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.key]; ok {
			continue
		}
		seen[entry.key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func selectConstraints(artifacts []artifact.Artifact, lookup RunTimeLookup, fallback time.Time) []artifactEntry {
	entries := make([]artifactEntry, 0, len(artifacts))
	for _, art := range artifacts {
		if art.Type != artifact.ArtifactConstraint {
			continue
		}
		if len(art.SupersededBy) > 0 {
			continue
		}
		key := normalizeStatement(art.Content.Text)
		if scope := strings.TrimSpace(art.Content.Scope); scope != "" {
			key = key + ":" + strings.ToLower(scope)
		}
		if key == "" {
			continue
		}
		entries = append(entries, artifactEntry{art: art, ts: resolveTime(art, lookup, fallback), key: key})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].ts.Equal(entries[j].ts) {
			return entries[i].ts.After(entries[j].ts)
		}
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return entries[i].art.ArtifactID < entries[j].art.ArtifactID
	})

	seen := make(map[string]struct{}, len(entries))
	out := make([]artifactEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.key]; ok {
			continue
		}
		seen[entry.key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func selectOpenThreads(artifacts []artifact.Artifact, lookup RunTimeLookup, fallback time.Time) []artifactEntry {
	entries := make([]artifactEntry, 0, len(artifacts))
	for _, art := range artifacts {
		if art.Type != artifact.ArtifactOpenThread {
			continue
		}
		key := normalizeStatement(art.Content.Text)
		if key == "" {
			continue
		}
		entries = append(entries, artifactEntry{art: art, ts: resolveTime(art, lookup, fallback), key: key})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].ts.Equal(entries[j].ts) {
			return entries[i].ts.After(entries[j].ts)
		}
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return entries[i].art.ArtifactID < entries[j].art.ArtifactID
	})

	seen := make(map[string]struct{}, len(entries))
	out := make([]artifactEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.key]; ok {
			continue
		}
		seen[entry.key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func selectOutcomes(artifacts []artifact.Artifact, lookup RunTimeLookup, fallback time.Time) []artifactEntry {
	byTask := make(map[string]artifactEntry)
	for _, art := range artifacts {
		if art.Type != artifact.ArtifactOutcome {
			continue
		}
		if !hasOutcomeContent(art) {
			continue
		}
		key := outcomeKey(art)
		entry := artifactEntry{art: art, ts: resolveTime(art, lookup, fallback), key: key}
		if existing, ok := byTask[key]; ok {
			if entry.ts.After(existing.ts) || (entry.ts.Equal(existing.ts) && entry.art.ArtifactID < existing.art.ArtifactID) {
				byTask[key] = entry
			}
			continue
		}
		byTask[key] = entry
	}

	entries := make([]artifactEntry, 0, len(byTask))
	for _, entry := range byTask {
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].ts.Equal(entries[j].ts) {
			return entries[i].ts.After(entries[j].ts)
		}
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return entries[i].art.ArtifactID < entries[j].art.ArtifactID
	})
	return entries
}

func outcomeKey(art artifact.Artifact) string {
	if art.TaskID != "" {
		return art.TaskID
	}
	if art.RunID != "" {
		return art.RunID
	}
	return art.ArtifactID
}

func hasOutcomeContent(art artifact.Artifact) bool {
	content := art.Content
	if strings.TrimSpace(content.Status) != "" {
		return true
	}
	if len(content.Summary) > 0 {
		return true
	}
	if len(content.Files) > 0 {
		return true
	}
	if len(content.Errors) > 0 {
		return true
	}
	return false
}

func resolveTime(art artifact.Artifact, lookup RunTimeLookup, fallback time.Time) time.Time {
	if lookup != nil {
		if ts, ok := lookup(art.TaskID, art.RunID); ok {
			return ts
		}
	}
	return fallback
}

func combineEntries(groups ...[]artifactEntry) []artifactEntry {
	combined := []artifactEntry{}
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, entry := range group {
			id := strings.TrimSpace(entry.art.ArtifactID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			combined = append(combined, entry)
		}
	}

	sort.SliceStable(combined, func(i, j int) bool {
		if !combined[i].ts.Equal(combined[j].ts) {
			return combined[i].ts.After(combined[j].ts)
		}
		return combined[i].art.ArtifactID < combined[j].art.ArtifactID
	})

	return combined
}

func buildSection(entries []artifactEntry, budget int, totalRemaining *int, formatter func(artifactEntry) string) (Section, int) {
	remaining := budget
	section := Section{}
	for _, entry := range entries {
		line := strings.TrimSpace(formatter(entry))
		if line == "" {
			continue
		}
		lineTokens := estimateTokens(line)
		sectionRemaining := remaining
		if sectionRemaining <= 0 {
			continue
		}
		allowed := sectionRemaining
		if totalRemaining != nil && *totalRemaining >= 0 && *totalRemaining < allowed {
			allowed = *totalRemaining
		}
		if allowed <= 0 {
			continue
		}
		if lineTokens > allowed {
			line = truncateTokens(line, allowed)
			lineTokens = estimateTokens(line)
		}
		if lineTokens == 0 || lineTokens > allowed {
			continue
		}
		section.Items = append(section.Items, line)
		section.Tokens += lineTokens
		remaining -= lineTokens
		if totalRemaining != nil {
			*totalRemaining -= lineTokens
		}
	}
	return section, section.Tokens
}

func formatDecision(entry artifactEntry) string {
	text := strings.TrimSpace(entry.art.Content.Text)
	if text == "" {
		return ""
	}
	if rationale := strings.TrimSpace(entry.art.Content.Rationale); rationale != "" {
		text = fmt.Sprintf("%s (rationale: %s)", text, rationale)
	}
	if id := strings.TrimSpace(entry.art.ArtifactID); id != "" {
		text = fmt.Sprintf("%s [id: %s]", text, id)
	}
	return text
}

func formatConstraint(entry artifactEntry) string {
	text := strings.TrimSpace(entry.art.Content.Text)
	if text == "" {
		return ""
	}
	if rationale := strings.TrimSpace(entry.art.Content.Rationale); rationale != "" {
		text = fmt.Sprintf("%s (rationale: %s)", text, rationale)
	}
	if id := strings.TrimSpace(entry.art.ArtifactID); id != "" {
		text = fmt.Sprintf("%s [id: %s]", text, id)
	}
	return text
}

func formatOpenThread(entry artifactEntry) string {
	text := strings.TrimSpace(entry.art.Content.Text)
	if text == "" {
		return ""
	}
	if id := strings.TrimSpace(entry.art.ArtifactID); id != "" {
		text = fmt.Sprintf("%s [id: %s]", text, id)
	}
	return text
}

func formatOutcome(entry artifactEntry) string {
	label := strings.TrimSpace(entry.art.TaskID)
	if label == "" {
		label = strings.TrimSpace(entry.art.RunID)
	}
	if label == "" {
		label = "unknown-task"
	}
	line := label
	if status := strings.TrimSpace(entry.art.Content.Status); status != "" {
		line = fmt.Sprintf("%s (%s)", line, status)
	}
	summary := outcomeSummary(entry.art.Content)
	if summary != "" {
		line = fmt.Sprintf("%s: %s", line, summary)
	}
	if id := strings.TrimSpace(entry.art.ArtifactID); id != "" {
		line = fmt.Sprintf("%s [id: %s]", line, id)
	}
	return line
}

func outcomeSummary(content artifact.Content) string {
	if len(content.Summary) > 0 {
		return strings.Join(limitStrings(content.Summary, 3), "; ")
	}
	if len(content.Errors) > 0 {
		return strings.TrimSpace(content.Errors[0])
	}
	if len(content.Files) > 0 {
		return "Files: " + strings.Join(limitStrings(content.Files, 3), ", ")
	}
	return ""
}

func formatArtifactPointer(entry artifactEntry) string {
	id := strings.TrimSpace(entry.art.ArtifactID)
	if id == "" {
		return ""
	}
	typeLabel := strings.TrimSpace(string(entry.art.Type))
	if typeLabel == "" {
		return id
	}
	return fmt.Sprintf("%s (%s)", id, typeLabel)
}

func goalLine(goal string) string {
	if strings.TrimSpace(goal) == "" {
		return ""
	}
	return "Session Goal: " + strings.TrimSpace(goal)
}

func estimateTokens(text string) int {
	return len(strings.Fields(text))
}

func truncateTokens(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	fields := strings.Fields(text)
	if len(fields) <= limit {
		return text
	}
	fields = fields[:limit]
	fields[len(fields)-1] = fields[len(fields)-1] + "..."
	return strings.Join(fields, " ")
}

func normalizeStatement(statement string) string {
	statement = strings.TrimSpace(statement)
	statement = strings.Trim(statement, ".;:")
	statement = strings.ToLower(statement)
	return strings.Join(strings.Fields(statement), " ")
}

func limitStrings(values []string, max int) []string {
	if len(values) <= max {
		return values
	}
	return values[:max]
}
