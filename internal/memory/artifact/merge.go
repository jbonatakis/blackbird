package artifact

import (
	"fmt"
	"strings"
)

type artifactIndex struct {
	byID       map[string]int
	byDedupKey map[ArtifactType]map[string]int
	byTopicKey map[ArtifactType]map[string]int
}

func MergeArtifacts(existing []Artifact, incoming []Artifact) []Artifact {
	result := make([]Artifact, 0, len(existing)+len(incoming))
	for _, art := range existing {
		result = append(result, normalizeArtifact(art))
	}
	index := newArtifactIndex(result)

	for _, art := range incoming {
		art = normalizeArtifact(art)
		if idx, ok := index.byID[art.ArtifactID]; ok {
			mergeProvenance(&result[idx], art.Provenance)
			continue
		}

		switch art.Type {
		case ArtifactDecision, ArtifactConstraint:
			if key := dedupKey(art); key != "" {
				if idx, ok := index.byDedupKey[art.Type][key]; ok {
					mergeProvenance(&result[idx], art.Provenance)
					continue
				}
			}
			if topic := topicKey(art); topic != "" {
				if idx, ok := index.byTopicKey[art.Type][topic]; ok {
					prev := &result[idx]
					if !contains(prev.SupersededBy, art.ArtifactID) {
						prev.SupersededBy = append(prev.SupersededBy, art.ArtifactID)
					}
					art.Supersedes = appendUnique(art.Supersedes, prev.ArtifactID)
				}
			}
		}

		result = append(result, art)
		index.add(art, len(result)-1)
	}
	return result
}

func newArtifactIndex(artifacts []Artifact) *artifactIndex {
	index := &artifactIndex{
		byID:       make(map[string]int),
		byDedupKey: map[ArtifactType]map[string]int{},
		byTopicKey: map[ArtifactType]map[string]int{},
	}
	for i, art := range artifacts {
		index.add(art, i)
	}
	return index
}

func (idx *artifactIndex) add(art Artifact, index int) {
	if art.ArtifactID != "" {
		idx.byID[art.ArtifactID] = index
	}
	if art.Type == ArtifactDecision || art.Type == ArtifactConstraint {
		if key := dedupKey(art); key != "" {
			if idx.byDedupKey[art.Type] == nil {
				idx.byDedupKey[art.Type] = map[string]int{}
			}
			idx.byDedupKey[art.Type][key] = index
		}
		if topic := topicKey(art); topic != "" {
			if idx.byTopicKey[art.Type] == nil {
				idx.byTopicKey[art.Type] = map[string]int{}
			}
			idx.byTopicKey[art.Type][topic] = index
		}
	}
}

func normalizeArtifact(art Artifact) Artifact {
	if art.SchemaVersion == 0 {
		art.SchemaVersion = SchemaVersion
	}
	if art.BuilderVersion == "" {
		art.BuilderVersion = BuilderVersion
	}
	return art
}

func dedupKey(art Artifact) string {
	text := normalizeStatement(art.Content.Text)
	if text == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s", art.Type, scopeKey(art), text)
}

func topicKey(art Artifact) string {
	text := normalizeStatement(art.Content.Text)
	if text == "" {
		return ""
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return scopeKey(art) + ":" + parts[0]
	}
	return scopeKey(art) + ":" + parts[0] + " " + parts[1]
}

func scopeKey(art Artifact) string {
	if art.SessionID != "" || art.TaskID != "" {
		return art.SessionID + "/" + art.TaskID
	}
	if art.RunID != "" {
		return art.RunID
	}
	return "global"
}

func mergeProvenance(target *Artifact, incoming []Provenance) {
	if target == nil || len(incoming) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(target.Provenance))
	for _, prov := range target.Provenance {
		seen[provenanceKey(prov)] = struct{}{}
	}
	for _, prov := range incoming {
		key := provenanceKey(prov)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		target.Provenance = append(target.Provenance, prov)
	}
}

func provenanceKey(prov Provenance) string {
	return fmt.Sprintf("%d:%s:%d:%d", prov.ItemIndex, prov.ItemType, prov.ContentStart, prov.ContentEnd)
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
