package claudestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJSONL(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFullParseAggregates(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"user","sessionId":"abc","cwd":"/home/proj","gitBranch":"main","timestamp":"2026-06-12T10:00:00Z"}`,
		`{"type":"assistant","timestamp":"2026-06-12T10:01:00Z","message":{"model":"claude-opus-4-8","stop_reason":"end_turn"}}`,
		`{"type":"ai-title","aiTitle":"generated title"}`,
	)
	rec := fullParse(path)

	if rec.SessionID != "abc" {
		t.Errorf("SessionID = %q", rec.SessionID)
	}
	if rec.CWD != "/home/proj" {
		t.Errorf("CWD = %q", rec.CWD)
	}
	if rec.GitBranch != "main" {
		t.Errorf("GitBranch = %q", rec.GitBranch)
	}
	if rec.Model != "claude-opus-4-8" {
		t.Errorf("Model = %q", rec.Model)
	}
	if rec.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", rec.MessageCount)
	}
	if rec.Title != "generated title" {
		t.Errorf("Title = %q", rec.Title)
	}
	if !rec.Awaiting {
		t.Error("Awaiting should be true after end_turn")
	}
	if rec.FirstTSMs == 0 || rec.LastTSMs <= rec.FirstTSMs {
		t.Errorf("timestamps wrong: first=%d last=%d", rec.FirstTSMs, rec.LastTSMs)
	}
}

func TestFullParseSkipsBadLines(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"user","sessionId":"abc","timestamp":"2026-06-12T10:00:00Z"}`,
		``,
		`   `,
		`not json at all`,
		`{"type":"assistant","message":{"model":"claude-opus-4-8","stop_reason":"end_turn"}}`,
	)
	rec := fullParse(path)
	if rec.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2 (bad lines skipped)", rec.MessageCount)
	}
}

func TestAwaitingResetByUserTurn(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"assistant","message":{"model":"claude-opus-4-8","stop_reason":"end_turn"}}`,
		`{"type":"user","sessionId":"abc","timestamp":"2026-06-12T10:00:00Z"}`,
	)
	rec := fullParse(path)
	if rec.Awaiting {
		t.Error("Awaiting should be reset to false by a trailing user turn")
	}
}

func TestSyntheticAssistantIgnored(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"assistant","message":{"model":"claude-opus-4-8","stop_reason":"end_turn"}}`,
		`{"type":"assistant","message":{"model":"<synthetic>","stop_reason":"stop"}}`,
	)
	rec := fullParse(path)
	if rec.Model != "claude-opus-4-8" {
		t.Errorf("Model = %q, synthetic line must not clobber model", rec.Model)
	}
	if !rec.Awaiting {
		t.Error("synthetic line must not clobber the end_turn waiting signal")
	}
}

func TestSidechainDetected(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"user","sessionId":"abc","isSidechain":true,"timestamp":"2026-06-12T10:00:00Z"}`,
	)
	rec := fullParse(path)
	if !rec.IsSidechain {
		t.Error("IsSidechain not detected")
	}
}

func TestTitlePrecedence(t *testing.T) {
	cases := []struct {
		name string
		rec  record
		want string
	}{
		{"custom over ai over preview", record{Title: "ai", CustomTitle: "custom", Preview: "prompt"}, "custom"},
		{"ai over preview", record{Title: "ai", Preview: "prompt"}, "ai"},
		{"preview fallback", record{Preview: "prompt"}, "prompt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := recordToSession("/p/x.jsonl", c.rec)
			if s.Title != c.want {
				t.Errorf("Title = %q, want %q", s.Title, c.want)
			}
		})
	}
}

func TestRecordToSessionIDFallback(t *testing.T) {
	rec := record{} // no SessionID
	s := recordToSession("/some/dir/deadbeef-1234.jsonl", rec)
	if s.ID != "deadbeef-1234" {
		t.Errorf("ID = %q, want deadbeef-1234 (derived from path)", s.ID)
	}
}

func TestParseIntoAppend(t *testing.T) {
	path := writeJSONL(t,
		`{"type":"user","sessionId":"abc","timestamp":"2026-06-12T10:00:00Z"}`,
	)
	info, _ := os.Stat(path)
	firstSize := info.Size()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"type":"assistant","message":{"model":"claude-opus-4-8","stop_reason":"end_turn"}}` + "\n")
	f.Close()

	var rec record
	n1, err := parseInto(&rec, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Re-parse only the appended tail into a fresh record to confirm offset seek.
	var tail record
	n2, err := parseInto(&tail, path, firstSize)
	if err != nil {
		t.Fatal(err)
	}
	if n1 != 2 {
		t.Errorf("full parse lines = %d, want 2", n1)
	}
	if n2 != 1 {
		t.Errorf("tail parse lines = %d, want 1", n2)
	}
	if tail.Model != "claude-opus-4-8" {
		t.Errorf("tail Model = %q", tail.Model)
	}
}
