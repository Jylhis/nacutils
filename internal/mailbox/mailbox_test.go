package mailbox_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

func tmpBase(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func makeEnvelope(t *testing.T, sender, recipient string) *envelope.Envelope {
	t.Helper()
	e, err := envelope.New(sender, recipient, envelope.KindNote, "subj", "body")
	if err != nil {
		t.Fatalf("envelope.New: %v", err)
	}
	return e
}

func TestAppendAndReadAll(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")

	e1 := makeEnvelope(t, "bob", "alice")
	e2 := makeEnvelope(t, "carol", "alice")

	if err := mailbox.Append(dir, e1); err != nil {
		t.Fatalf("Append e1: %v", err)
	}
	if err := mailbox.Append(dir, e2); err != nil {
		t.Fatalf("Append e2: %v", err)
	}

	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("ReadAll: got %d envelopes, want 2", len(got))
	}
}

func TestAppend_Idempotent(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")
	e := makeEnvelope(t, "bob", "alice")

	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("second Append (idempotent): %v", err)
	}

	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("idempotent: got %d envelopes, want 1", len(got))
	}
}

func TestReadAll_Missing(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "nobody")
	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll on missing dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice for missing inbox, got %d", len(got))
	}
}

func TestFindByID(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")
	e := makeEnvelope(t, "bob", "alice")
	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, recipDir, err := mailbox.FindByID(base, e.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("FindByID: got ID %q, want %q", got.ID, e.ID)
	}
	wantDir := filepath.Join(base, "alice")
	if recipDir != wantDir {
		t.Errorf("FindByID recipDir: got %q, want %q", recipDir, wantDir)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	base := tmpBase(t)
	_, _, err := mailbox.FindByID(base, "does-not-exist")
	if err == nil {
		t.Error("expected error for missing envelope")
	}
}

func TestRemove(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")
	e1 := makeEnvelope(t, "bob", "alice")
	e2 := makeEnvelope(t, "carol", "alice")

	if err := mailbox.Append(dir, e1); err != nil {
		t.Fatalf("Append e1: %v", err)
	}
	if err := mailbox.Append(dir, e2); err != nil {
		t.Fatalf("Append e2: %v", err)
	}

	if err := mailbox.Remove(dir, e1.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll after Remove: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("after Remove: got %d envelopes, want 1", len(got))
	}
	if got[0].ID != e2.ID {
		t.Errorf("remaining envelope: got %q, want %q", got[0].ID, e2.ID)
	}
}

func TestRemove_NotFound(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Create inbox file with one envelope
	e := makeEnvelope(t, "bob", "alice")
	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := mailbox.Remove(dir, "nonexistent-id"); err == nil {
		t.Error("expected error removing nonexistent envelope")
	}
}

func TestBaseDir(t *testing.T) {
	// With XDG_DATA_HOME set
	t.Setenv("XDG_DATA_HOME", "/tmp/testxdg")
	got := mailbox.BaseDir()
	want := "/tmp/testxdg/nacutils/mail"
	if got != want {
		t.Errorf("BaseDir with XDG_DATA_HOME: got %q, want %q", got, want)
	}
}

func TestMarkRead(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")
	e := makeEnvelope(t, "bob", "alice")

	if err := mailbox.Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	readAt := time.Date(2026, 6, 2, 7, 0, 0, 0, time.UTC)
	if err := mailbox.MarkRead(base, e.ID, readAt); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if got[0].Meta["read_at"] != readAt.Format(time.RFC3339) {
		t.Fatalf("read_at: got %v want %s", got[0].Meta["read_at"], readAt.Format(time.RFC3339))
	}
}

func TestCleanup_DryRunAndApply(t *testing.T) {
	base := tmpBase(t)
	dir := mailbox.RecipientDir(base, "alice")

	oldRead := makeEnvelope(t, "ops", "alice")
	oldRead.CreatedAt = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	oldRead.Meta["read_at"] = time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

	oldUnread := makeEnvelope(t, "ops", "alice")
	oldUnread.CreatedAt = time.Date(2026, 5, 1, 13, 0, 0, 0, time.UTC)

	newRead := makeEnvelope(t, "ops", "alice")
	newRead.CreatedAt = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	newRead.Meta["read_at"] = time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)

	for _, e := range []*envelope.Envelope{oldRead, oldUnread, newRead} {
		if err := mailbox.Append(dir, e); err != nil {
			t.Fatalf("Append %s: %v", e.ID, err)
		}
	}

	before := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	report, err := mailbox.Cleanup(base, "alice", before, false)
	if err != nil {
		t.Fatalf("Cleanup dry-run: %v", err)
	}
	if report.Eligible != 1 || report.Removed != 0 {
		t.Fatalf("dry-run report: eligible=%d removed=%d", report.Eligible, report.Removed)
	}

	got, err := mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll after dry-run: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("dry-run should keep all envelopes, got %d", len(got))
	}

	report, err = mailbox.Cleanup(base, "alice", before, true)
	if err != nil {
		t.Fatalf("Cleanup apply: %v", err)
	}
	if report.Removed != 1 {
		t.Fatalf("apply report removed=%d want 1", report.Removed)
	}

	got, err = mailbox.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll after apply: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("apply should keep 2 envelopes, got %d", len(got))
	}
	for _, e := range got {
		if e.ID == oldRead.ID {
			t.Fatal("old read envelope was not removed")
		}
	}
}

func TestCleanup_NoMatchingRecipients(t *testing.T) {
	base := tmpBase(t)

	report, err := mailbox.Cleanup(base, "missing", time.Now().UTC(), true)
	if err != nil {
		t.Fatalf("Cleanup missing recipient: %v", err)
	}
	if report.MatchedRecipients != 0 || report.Removed != 0 || report.Inspected != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCleanup_RecipientScoping(t *testing.T) {
	base := tmpBase(t)
	before := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	aliceDir := mailbox.RecipientDir(base, "alice")
	aliceEnvelope := makeEnvelope(t, "ops", "alice")
	aliceEnvelope.CreatedAt = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	aliceEnvelope.Meta["read_at"] = time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := mailbox.Append(aliceDir, aliceEnvelope); err != nil {
		t.Fatalf("Append alice: %v", err)
	}

	bobDir := mailbox.RecipientDir(base, "bob")
	bobEnvelope := makeEnvelope(t, "ops", "bob")
	bobEnvelope.CreatedAt = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	bobEnvelope.Meta["read_at"] = time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := mailbox.Append(bobDir, bobEnvelope); err != nil {
		t.Fatalf("Append bob: %v", err)
	}

	report, err := mailbox.Cleanup(base, "alice", before, true)
	if err != nil {
		t.Fatalf("Cleanup scoped: %v", err)
	}
	if report.MatchedRecipients != 1 || report.Removed != 1 {
		t.Fatalf("unexpected scoped report: %+v", report)
	}

	aliceGot, err := mailbox.ReadAll(aliceDir)
	if err != nil {
		t.Fatalf("ReadAll alice: %v", err)
	}
	if len(aliceGot) != 0 {
		t.Fatalf("alice should be empty, got %d", len(aliceGot))
	}

	bobGot, err := mailbox.ReadAll(bobDir)
	if err != nil {
		t.Fatalf("ReadAll bob: %v", err)
	}
	if len(bobGot) != 1 {
		t.Fatalf("bob should keep mail, got %d", len(bobGot))
	}
}
