package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/IngTian/witness/internal/store"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type failEmptyEmbedder struct{ t *testing.T }

func (e failEmptyEmbedder) Embed(string) ([]float32, error) {
	e.t.Fatal("empty search must not require an embedding model")
	return nil, nil
}

func TestToolsDiagnoseEmptyStages(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state store.ArchiveState
		want  string
	}{
		{"no capture", store.ArchiveState{}, "witness wire claude"},
		{"not mined", store.ArchiveState{HasRaw: true}, "No observations have been extracted"},
		{"not reviewed", store.ArchiveState{HasRaw: true, HasObservations: true, HasDerived: true}, "reviewer may not have run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeMCPStore{state: tc.state}
			ctx := context.Background()
			serverT, clientT := mcpsdk.NewInMemoryTransports()
			ss, err := newServer(fake, failEmptyEmbedder{t}, "test", nil).Connect(ctx, serverT, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientT, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			for _, tool := range []string{"get_facets", "get_profile", "search_observations"} {
				args := map[string]any{}
				if tool == "search_observations" {
					args["query"] = "learning"
				}
				res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: tool, Arguments: args})
				if err != nil || res.IsError {
					t.Fatalf("%s: %v, %+v", tool, err, res)
				}
				msg := res.Content[0].(*mcpsdk.TextContent).Text
				want := tc.want
				if tool == "search_observations" && tc.state.HasObservations {
					want = "none carry embeddings"
				}
				if !strings.Contains(msg, want) {
					t.Errorf("%s: %q missing %q", tool, msg, want)
				}
				if !tc.state.HasObservations && strings.Contains(msg, "reviewer") {
					t.Errorf("%s blames reviewer: %s", tool, msg)
				}
			}
		})
	}
}
