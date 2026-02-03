package index

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

var relatedWeights = map[string]float64{
	linkTypeRun:       3.0,
	linkTypeTask:      2.0,
	linkTypeProvItem:  2.5,
	linkTypeTraceSpan: 2.2,
	linkTypeTraceReq:  2.0,
	linkTypeSession:   1.0,
}

// Related returns artifacts linked by run/task/provenance adjacency.
func (idx *Index) Related(artifactID string, opts RelatedOptions) ([]SearchCard, error) {
	if idx == nil || idx.db == nil {
		return nil, fmt.Errorf("index not initialized")
	}
	if strings.TrimSpace(artifactID) == "" {
		return nil, fmt.Errorf("artifact id required")
	}
	norm := opts.normalized()

	art, found, err := idx.Get(artifactID)
	if err != nil {
		return nil, err
	}
	if !found {
		return []SearchCard{}, nil
	}
	links := linkKeys(art)
	if len(links) == 0 {
		return []SearchCard{}, nil
	}

	linkQuery, args := buildLinkQuery(links)
	rows, err := idx.db.QueryContext(context.Background(), linkQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("related query: %w", err)
	}
	defer rows.Close()

	scores := make(map[string]relatedScore)
	for rows.Next() {
		var otherID, linkType string
		if err := rows.Scan(&otherID, &linkType); err != nil {
			return nil, fmt.Errorf("scan related row: %w", err)
		}
		if otherID == art.ArtifactID {
			continue
		}
		entry := scores[otherID]
		if linkType == linkTypeRun {
			entry.hasRun = true
		} else if linkType == linkTypeTask {
			entry.hasTask = true
		}
		weight := relatedWeights[linkType]
		if weight == 0 {
			weight = 1.0
		}
		entry.score += weight
		scores[otherID] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("related rows: %w", err)
	}
	if len(scores) == 0 {
		return []SearchCard{}, nil
	}

	ordered := sortRelated(scores)
	if len(ordered) > norm.Limit {
		ordered = ordered[:norm.Limit]
	}

	cards, err := idx.loadCardsByIDs(ordered, norm.SnippetMaxLen)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func buildLinkQuery(links map[linkKey]struct{}) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(links)*2)
	sb.WriteString("SELECT artifact_id, link_type FROM artifact_links WHERE ")
	first := true
	for link := range links {
		if !first {
			sb.WriteString(" OR ")
		}
		first = false
		sb.WriteString("(link_type = ? AND link_value = ?)")
		args = append(args, link.linkType, link.value)
	}
	return sb.String(), args
}

type relatedItem struct {
	id      string
	score   float64
	hasRun  bool
	hasTask bool
}

type relatedScore struct {
	score   float64
	hasRun  bool
	hasTask bool
}

func sortRelated(scores map[string]relatedScore) []relatedItem {
	ordered := make([]relatedItem, 0, len(scores))
	for id, score := range scores {
		ordered = append(ordered, relatedItem{id: id, score: score.score, hasRun: score.hasRun, hasTask: score.hasTask})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].hasRun != ordered[j].hasRun {
			return ordered[i].hasRun
		}
		if ordered[i].hasTask != ordered[j].hasTask {
			return ordered[i].hasTask
		}
		if ordered[i].score == ordered[j].score {
			return ordered[i].id < ordered[j].id
		}
		return ordered[i].score > ordered[j].score
	})
	return ordered
}

func (idx *Index) loadCardsByIDs(ids []relatedItem, snippetMaxLen int) ([]SearchCard, error) {
	if len(ids) == 0 {
		return []SearchCard{}, nil
	}
	idArgs := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	for _, item := range ids {
		placeholders = append(placeholders, "?")
		idArgs = append(idArgs, item.id)
	}
	query := fmt.Sprintf(`SELECT id, session_id, task_id, run_id, type, created_at, provenance_json, text FROM artifacts WHERE id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := idx.db.QueryContext(context.Background(), query, idArgs...)
	if err != nil {
		return nil, fmt.Errorf("load cards: %w", err)
	}
	defer rows.Close()

	cards := make(map[string]SearchCard, len(ids))
	for rows.Next() {
		var (
			card        SearchCard
			createdUnix int64
			provRaw     string
			text        string
		)
		if err := rows.Scan(&card.ArtifactID, &card.SessionID, &card.TaskID, &card.RunID, &card.Type, &createdUnix, &provRaw, &text); err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}
		card.CreatedAt = time.Unix(createdUnix, 0)
		if provRaw != "" {
			if err := json.Unmarshal([]byte(provRaw), &card.Provenance); err != nil {
				return nil, fmt.Errorf("decode provenance: %w", err)
			}
		}
		card.Snippet = boundSnippet(text, snippetMaxLen)
		cards[card.ArtifactID] = card
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load cards rows: %w", err)
	}

	result := make([]SearchCard, 0, len(ids))
	for _, item := range ids {
		card, ok := cards[item.id]
		if !ok {
			continue
		}
		card.Score = item.score
		result = append(result, card)
	}
	return result, nil
}
