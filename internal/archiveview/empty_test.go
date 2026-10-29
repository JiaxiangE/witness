package archiveview

import (
	"strings"
	"testing"

	"github.com/IngTian/witness/internal/store"
)

func TestEmptyStagesAndRetainedHistory(t *testing.T) {
	t.Setenv("WITNESS_HOME", t.TempDir())
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	check := func(kind, lens, want, absent string) {
		t.Helper()
		msg, err := EmptyMessage(st, kind, lens)
		if err != nil || !strings.Contains(msg, want) || (absent != "" && strings.Contains(msg, absent)) {
			t.Fatalf("%s/%s: %q, %v", kind, lens, msg, err)
		}
	}
	for _, kind := range []string{"facets", "observations", "profile"} {
		check(kind, "default", "witness wire claude", "reviewer")
	}
	if err := st.AppendRaw(store.RawRecord{Session: "actual-session", TS: "2026-01-01T00:00:00Z", Role: "user", Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"facets", "observations", "profile"} {
		check(kind, "default", "No observations have been extracted", "reviewer")
	}
	if err := st.AppendObservations([]store.Observation{{ID: "o1", Lens: "default", Session: "actual-session", Observation: "learned"}}); err != nil {
		t.Fatal(err)
	}
	check("facets", "default", "reviewer may not have run", "No raw records")
	check("profile", store.ProfileUnified, "reviewer may not have run", "No raw records")
	check("facets", "other", "No observations have been extracted", "reviewer")
	check("observations", "default", "none carry embeddings", "reviewer")
	if err := st.WriteFacets([]store.Facet{{Lens: "default", Dimension: "thinking", Key: "learn", Versions: []store.FacetVersion{{Value: "learns"}}}}); err != nil {
		t.Fatal(err)
	}
	check("profile", store.ProfileUnified, "Facets exist", "reviewer")
	if _, _, err := st.PruneSessionsBefore("2027-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	check("facets", "default", "reviewer may not have run", "No raw records")
	check("facets", "other", "No observations have been extracted", "No raw records")
}
