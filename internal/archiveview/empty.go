// Package archiveview shares actionable empty states between the CLI and MCP.
package archiveview

import (
	"fmt"

	"github.com/IngTian/witness/internal/store"
)

type Reader interface {
	ArchiveState(lens string) (store.ArchiveState, error)
	ReadFacets() ([]store.Facet, error)
}

func EmptyMessage(st Reader, kind, lens string) (string, error) {
	filter := lens
	if kind == "profile" && lens == store.ProfileUnified {
		filter = ""
	}
	s, err := st.ArchiveState(filter)
	if err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("No %s found for lens %q. ", kind, lens)
	if !s.HasRaw && !s.HasDerived {
		return prefix + "No raw records have been captured. The MCP server alone reads the archive; it does not capture sessions. Run `witness wire claude`, add `@witness-ai/opencode` to the plugin array in opencode.json, or run `witness import --agent codex`. For other sources, use `witness ingest`.", nil
	}
	if kind == "profile" {
		facets, err := st.ReadFacets()
		if err != nil {
			return "", err
		}
		for _, f := range facets {
			if (filter == "" || f.Lens == filter) && f.Current() != nil {
				return prefix + "Facets exist, but the narrative is not available. Retry `witness profile` and check the configured distillation runner.", nil
			}
		}
	}
	if !s.HasObservations {
		return prefix + "No observations have been extracted for this lens yet. Check `witness status` and `witness lens list`; enable the lens if needed, then run `witness worker run`.", nil
	}
	if kind == "observations" {
		return prefix + "Observations exist, but none carry embeddings, so nothing is searchable. Check `witness doctor` for the embedder's status.", nil
	}
	return prefix + "Observations exist; the reviewer may not have run yet. Check `witness lens list` and run `witness worker review` for enabled lenses.", nil
}
