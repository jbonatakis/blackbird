package index

import (
	"fmt"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

type linkKey struct {
	linkType string
	value    string
}

const (
	linkTypeSession   = "session"
	linkTypeTask      = "task"
	linkTypeRun       = "run"
	linkTypeProvItem  = "prov_item"
	linkTypeTraceReq  = "trace_req"
	linkTypeTraceSpan = "trace_span"
)

func linkKeys(art artifact.Artifact) map[linkKey]struct{} {
	links := make(map[linkKey]struct{})
	add := func(linkType, value string) {
		if value == "" {
			return
		}
		links[linkKey{linkType: linkType, value: value}] = struct{}{}
	}

	add(linkTypeSession, art.SessionID)
	add(linkTypeTask, art.TaskID)
	add(linkTypeRun, art.RunID)

	for _, prov := range art.Provenance {
		if prov.ItemType != "" {
			add(linkTypeProvItem, fmt.Sprintf("%s:%d", prov.ItemType, prov.ItemIndex))
		}
		for _, span := range prov.Spans {
			for _, trace := range span.Trace {
				if trace.RequestID != "" {
					add(linkTypeTraceReq, trace.RequestID)
					add(linkTypeTraceSpan, fmt.Sprintf("%s:%d", trace.RequestID, trace.EventIndex))
				}
			}
		}
	}

	return links
}
