package planquality

import (
	"sort"
	"strings"
	"unicode"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// LeafTaskIDs returns all leaf task IDs in deterministic order.
func LeafTaskIDs(g plan.WorkGraph) []string {
	ids := make([]string, 0, len(g.Items))
	for taskID, it := range g.Items {
		if len(it.ChildIDs) != 0 {
			continue
		}
		ids = append(ids, taskID)
	}
	sort.Strings(ids)
	return ids
}

// NormalizeText lowercases text, strips punctuation, and collapses whitespace.
func NormalizeText(value string) string {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

// NormalizeTexts normalizes a list of strings and drops empty results.
func NormalizeTexts(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := NormalizeText(value)
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

// ContainsAnyNormalizedPhrase checks whether value contains any phrase after normalization.
func ContainsAnyNormalizedPhrase(value string, phrases []string) bool {
	normalizedValue := NormalizeText(value)
	if normalizedValue == "" {
		return false
	}

	bounded := " " + normalizedValue + " "
	for _, phrase := range phrases {
		normalizedPhrase := NormalizeText(phrase)
		if normalizedPhrase == "" {
			continue
		}
		if strings.Contains(bounded, " "+normalizedPhrase+" ") {
			return true
		}
	}

	return false
}
