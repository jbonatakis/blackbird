package index

import (
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

const (
	defaultLimit           = 10
	defaultSnippetTokens   = 16
	defaultSnippetMaxLen   = 180
	defaultCandidateFactor = 5
	defaultCandidateMin    = 50
	defaultCandidateMax    = 500
)

// SearchOptions configures a lexical search over the index.
type SearchOptions struct {
	Query           string
	SessionID       string
	TaskID          string
	RunID           string
	Types           []artifact.ArtifactType
	Limit           int
	Offset          int
	SnippetTokens   int
	SnippetMaxLen   int
	CandidateLimit  int
	Now             time.Time
	RecencyHalfLife time.Duration
	TypeWeights     map[artifact.ArtifactType]float64
}

// SearchCard is a bounded result returned by search/related.
type SearchCard struct {
	ArtifactID string
	SessionID  string
	TaskID     string
	RunID      string
	Type       artifact.ArtifactType
	Snippet    string
	Score      float64
	Provenance []artifact.Provenance
	CreatedAt  time.Time
}

// RelatedOptions configures related artifact lookup.
type RelatedOptions struct {
	Limit         int
	SnippetMaxLen int
}

// RebuildOptions configures index rebuild behavior.
type RebuildOptions struct {
	Now           time.Time
	TimestampFor  func(artifact.Artifact) time.Time
	RunTimeLookup RunTimeLookup
}

// RunTimeLookup returns the run timestamp for a task/run pair.
type RunTimeLookup func(taskID, runID string) (time.Time, bool)

func defaultTypeWeights() map[artifact.ArtifactType]float64 {
	return map[artifact.ArtifactType]float64{
		artifact.ArtifactOutcome:    1.2,
		artifact.ArtifactDecision:   1.4,
		artifact.ArtifactConstraint: 1.2,
		artifact.ArtifactOpenThread: 1.0,
		artifact.ArtifactTranscript: 0.6,
	}
}

func (opts SearchOptions) normalized() SearchOptions {
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}
	if opts.SnippetTokens <= 0 {
		opts.SnippetTokens = defaultSnippetTokens
	}
	if opts.SnippetMaxLen <= 0 {
		opts.SnippetMaxLen = defaultSnippetMaxLen
	}
	if opts.CandidateLimit <= 0 {
		candidate := (opts.Limit + opts.Offset) * defaultCandidateFactor
		if candidate < defaultCandidateMin {
			candidate = defaultCandidateMin
		}
		if candidate > defaultCandidateMax {
			candidate = defaultCandidateMax
		}
		opts.CandidateLimit = candidate
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.TypeWeights == nil {
		opts.TypeWeights = defaultTypeWeights()
	}
	return opts
}

func (opts RelatedOptions) normalized() RelatedOptions {
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}
	if opts.SnippetMaxLen <= 0 {
		opts.SnippetMaxLen = defaultSnippetMaxLen
	}
	return opts
}
