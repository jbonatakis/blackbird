package index

import (
	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

// OpenForProject opens the index for a project root.
func OpenForProject(projectRoot string) (*Index, error) {
	return Open(memory.IndexDBPath(projectRoot))
}

// SearchForProject opens the project index, executes the search, and closes it.
func SearchForProject(projectRoot string, opts SearchOptions) ([]SearchCard, error) {
	idx, err := OpenForProject(projectRoot)
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	return idx.Search(opts)
}

// GetForProject loads a full artifact from the project index.
func GetForProject(projectRoot, artifactID string) (artifact.Artifact, bool, error) {
	idx, err := OpenForProject(projectRoot)
	if err != nil {
		return artifact.Artifact{}, false, err
	}
	defer idx.Close()
	return idx.Get(artifactID)
}

// RelatedForProject loads related cards for an artifact from the project index.
func RelatedForProject(projectRoot, artifactID string, opts RelatedOptions) ([]SearchCard, error) {
	idx, err := OpenForProject(projectRoot)
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	return idx.Related(artifactID, opts)
}
