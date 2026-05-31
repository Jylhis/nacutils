package mailbox_test

import (
	"os"
	"path/filepath"
	"testing"

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
