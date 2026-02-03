package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

// Search performs an FTS search over indexed artifacts.
func (idx *Index) Search(opts SearchOptions) ([]SearchCard, error) {
	if idx == nil || idx.db == nil {
		return nil, fmt.Errorf("index not initialized")
	}
	if strings.TrimSpace(opts.Query) == "" {
		return nil, fmt.Errorf("query required")
	}
	norm := opts.normalized()

	query, args := buildSearchQuery(norm)
	rows, err := idx.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	cards := make([]SearchCard, 0, norm.CandidateLimit)
	for rows.Next() {
		card, bm25Score, text, err := scanSearchRow(rows)
		if err != nil {
			return nil, err
		}
		card.Snippet = boundSnippet(card.Snippet, norm.SnippetMaxLen)
		if card.Snippet == "" {
			card.Snippet = boundSnippet(text, norm.SnippetMaxLen)
		}
		card.Score = scoreResult(bm25Score, card.Type, card.CreatedAt, norm)
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}

	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].Score == cards[j].Score {
			return cards[i].CreatedAt.After(cards[j].CreatedAt)
		}
		return cards[i].Score > cards[j].Score
	})

	start := norm.Offset
	if start > len(cards) {
		return []SearchCard{}, nil
	}
	end := start + norm.Limit
	if end > len(cards) {
		end = len(cards)
	}
	return cards[start:end], nil
}

func buildSearchQuery(opts SearchOptions) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, 8)

	sb.WriteString(`SELECT artifacts.id, artifacts.session_id, artifacts.task_id, artifacts.run_id, artifacts.type, artifacts.created_at,`)
	sb.WriteString(` artifacts.provenance_json, artifacts.text,`)
	sb.WriteString(` snippet(artifacts_fts, 0, '[', ']', '...', ?) AS snippet,`)
	sb.WriteString(` bm25(artifacts_fts) AS bm25`)
	sb.WriteString(` FROM artifacts_fts JOIN artifacts ON artifacts_fts.rowid = artifacts.rowid`)
	sb.WriteString(` WHERE artifacts_fts MATCH ?`)

	args = append(args, opts.SnippetTokens)
	args = append(args, opts.Query)

	if opts.SessionID != "" {
		sb.WriteString(" AND artifacts.session_id = ?")
		args = append(args, opts.SessionID)
	}
	if opts.TaskID != "" {
		sb.WriteString(" AND artifacts.task_id = ?")
		args = append(args, opts.TaskID)
	}
	if opts.RunID != "" {
		sb.WriteString(" AND artifacts.run_id = ?")
		args = append(args, opts.RunID)
	}
	if len(opts.Types) > 0 {
		sb.WriteString(" AND artifacts.type IN (")
		for i, t := range opts.Types {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("?")
			args = append(args, string(t))
		}
		sb.WriteString(")")
	}

	sb.WriteString(" LIMIT ?")
	args = append(args, opts.CandidateLimit)

	return sb.String(), args
}

func scanSearchRow(rows *sql.Rows) (SearchCard, float64, string, error) {
	var (
		card          SearchCard
		createdUnix   int64
		provenanceRaw string
		text          string
		bm25Score     float64
	)
	if err := rows.Scan(
		&card.ArtifactID,
		&card.SessionID,
		&card.TaskID,
		&card.RunID,
		&card.Type,
		&createdUnix,
		&provenanceRaw,
		&text,
		&card.Snippet,
		&bm25Score,
	); err != nil {
		return SearchCard{}, 0, "", fmt.Errorf("scan search row: %w", err)
	}
	card.CreatedAt = time.Unix(createdUnix, 0)
	if provenanceRaw != "" {
		if err := json.Unmarshal([]byte(provenanceRaw), &card.Provenance); err != nil {
			return SearchCard{}, 0, "", fmt.Errorf("decode provenance: %w", err)
		}
	}
	return card, bm25Score, text, nil
}

func scoreResult(bm25Score float64, artType artifact.ArtifactType, createdAt time.Time, opts SearchOptions) float64 {
	textScore := 1.0
	if bm25Score >= 0 {
		textScore = 1.0 / (1.0 + bm25Score)
	}
	weight := 1.0
	if typeWeight, ok := opts.TypeWeights[artType]; ok {
		weight *= typeWeight
	}
	if opts.RecencyHalfLife > 0 {
		age := opts.Now.Sub(createdAt)
		if age < 0 {
			age = 0
		}
		weight *= math.Pow(0.5, float64(age)/float64(opts.RecencyHalfLife))
	}
	return textScore * weight
}
