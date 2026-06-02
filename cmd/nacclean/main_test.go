package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

func runCleanCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func writeEnvelope(t *testing.T, recipient string, createdAt time.Time, read bool) *envelope.Envelope {
	t.Helper()
	base := mailbox.BaseDir()
	dir := mailbox.RecipientDir(base, recipient)

	e, err := envelope.New("tester", recipient, envelope.KindNote, "subj", "body")
	if err != nil {
		t.Fatalf("envelope.New: %v", err)
	}
	e.CreatedAt = createdAt.UTC()
	if read {
		e.Meta["read_at"] = createdAt.Add(time.Hour).UTC().Format(time.RFC3339)
	}
	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	return e
}

func TestParseBefore(t *testing.T) {
	now := time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC)

	got, err := parseBefore("48h", now)
	if err != nil {
		t.Fatalf("duration parse: %v", err)
	}
	if want := now.Add(-48 * time.Hour); !got.Equal(want) {
		t.Fatalf("duration parse: got %s want %s", got, want)
	}

	got, err = parseBefore("2026-05-01T12:00:00Z", now)
	if err != nil {
		t.Fatalf("RFC3339 parse: %v", err)
	}
	if want := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("RFC3339 parse: got %s want %s", got, want)
	}

	got, err = parseBefore("1746100800000", now)
	if err != nil {
		t.Fatalf("unix-ms parse: %v", err)
	}
	if want := time.UnixMilli(1746100800000).UTC(); !got.Equal(want) {
		t.Fatalf("unix-ms parse: got %s want %s", got, want)
	}
}

func TestNacclean_DryRun_NoOp(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	oldRead := writeEnvelope(t, "alice", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), true)
	writeEnvelope(t, "alice", time.Date(2026, 5, 1, 13, 0, 0, 0, time.UTC), false)

	out, err := runCleanCmd(t, "alice", "--before", "2026-05-15T00:00:00Z")
	if err != nil {
		t.Fatalf("nacclean dry-run: %v", err)
	}
	if !strings.Contains(out, "removed=0") || !strings.Contains(out, "eligible=1") || !strings.Contains(out, "dry_run=true") {
		t.Fatalf("unexpected dry-run output: %s", out)
	}

	envelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "alice"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(envelopes) != 2 {
		t.Fatalf("dry-run should keep envelopes, got %d", len(envelopes))
	}
	found := false
	for _, e := range envelopes {
		if e.ID == oldRead.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("dry-run removed eligible envelope")
	}
}

func TestNacclean_RecipientScopingAndApply(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	alice := writeEnvelope(t, "alice", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), true)
	writeEnvelope(t, "bob", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), true)

	out, err := runCleanCmd(t, "alice", "--before", "2026-05-15T00:00:00Z", "--apply", "--json")
	if err != nil {
		t.Fatalf("nacclean apply: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json output: %v\n%s", err, out)
	}
	if payload["removed"].(float64) != 1 || payload["matched_recipients"].(float64) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	aliceEnvelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "alice"))
	if err != nil {
		t.Fatalf("ReadAll alice: %v", err)
	}
	if len(aliceEnvelopes) != 0 {
		t.Fatalf("alice mailbox should be empty, got %d", len(aliceEnvelopes))
	}

	bobEnvelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "bob"))
	if err != nil {
		t.Fatalf("ReadAll bob: %v", err)
	}
	if len(bobEnvelopes) != 1 {
		t.Fatalf("bob mailbox should remain untouched, got %d", len(bobEnvelopes))
	}
	if alice.ID == bobEnvelopes[0].ID {
		t.Fatal("cleanup removed the wrong recipient envelope")
	}
}

func TestNacclean_NoDeletionWhenNoRecipientsMatch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	out, err := runCleanCmd(t, "nobody", "--before", "48h")
	if err != nil {
		t.Fatalf("nacclean missing recipient: %v", err)
	}
	if !strings.Contains(out, "matched_recipients=0") || !strings.Contains(out, "removed=0") {
		t.Fatalf("unexpected no-match output: %s", out)
	}
}
