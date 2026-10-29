package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IngTian/witness/internal/store"
)

func TestCLIEmptyArchiveStages(t *testing.T) {
	t.Setenv("WITNESS_HOME", t.TempDir())
	t.Setenv("WITNESS_PROMPTS", filepath.Join("..", "..", "prompts"))
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := seedDefaultLens(st); err != nil {
		t.Fatal(err)
	}
	for stage := 0; stage < 3; stage++ {
		want := "No raw records have been captured"
		if stage == 1 {
			want = "No observations have been extracted"
		}
		if stage == 2 {
			want = "reviewer may not have run"
		}
		for name, call := range map[string]func() error{
			"facets":  func() error { return cmdFacets(nil, false) },
			"profile": func() error { return cmdProfile(nil, false) },
			"search":  func() error { return cmdObservationSearch("learn", "other", 8, false) },
		} {
			out, err := captureStdoutErr(t, call)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			expected := want
			if stage == 2 && name == "search" {
				expected = "No observations have been extracted"
			}
			if !strings.Contains(out, expected) {
				t.Errorf("stage %d %s: %s", stage, name, out)
			}
		}
		if stage == 0 {
			if err := st.AppendRaw(store.RawRecord{Session: "s", Role: "user", Text: "hello"}); err != nil {
				t.Fatal(err)
			}
		}
		if stage == 1 {
			if err := st.AppendObservations([]store.Observation{{ID: "o", Lens: "default", Observation: "learning"}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	out, err := captureStdoutErr(t, func() error { return cmdObservationSearch("query", "empty", 8, true) })
	if err != nil {
		t.Fatal(err)
	}
	var parsed observationSearchJSON
	if err := json.Unmarshal([]byte(out), &parsed); err != nil || parsed.Hits == nil || len(parsed.Hits) != 0 {
		t.Fatalf("JSON contract: %s, %v", out, err)
	}
}

func TestImportHookNeverBreaksSession(t *testing.T) {
	for _, payload := range []string{"{", `{}`, `{"session_id":"missing"}`} {
		out, err := captureStdoutErr(t, func() error { return cmdImportHook("codex", strings.NewReader(payload)) })
		if err != nil || strings.TrimSpace(out) != "{}" {
			t.Fatalf("hook: %q, %v", out, err)
		}
	}
}

func TestImportHookCapturesAndRetries(t *testing.T) {
	t.Setenv("WITNESS_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("WITNESS_CODEX_HOME", home)
	if err := os.Mkdir(filepath.Join(home, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	fixture := `{"type":"session_meta","payload":{"id":"hook-session"}}` + "\n" +
		`{"type":"response_item","timestamp":"2026-10-28T12:00:00Z","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello from Codex"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(home, "sessions", "rollout-hook-session.jsonl"), []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		out, err := captureStdoutErr(t, func() error {
			return cmdImportHook("codex", strings.NewReader(`{"session_id":"hook-session","hook_event_name":"Stop"}`))
		})
		if err != nil || strings.TrimSpace(out) != "{}" {
			t.Fatalf("hook: %s, %v", out, err)
		}
	}
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	recs, err := st.ReadRaw("codex:hook-session")
	if err != nil || len(recs) != 1 || recs[0].Text != "hello from Codex" || st.SessionPlatform("codex:hook-session") != "codex" {
		t.Fatalf("captured %+v, %v", recs, err)
	}
}
