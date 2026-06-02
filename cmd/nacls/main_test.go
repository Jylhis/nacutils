package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

func runNaclsCmd(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf.String(), err
}

func writeEnvelope(t *testing.T, base, recipient, sender string, read bool) {
	t.Helper()

	e, err := envelope.New(sender, recipient, envelope.KindNote, "", "body")
	if err != nil {
		t.Fatalf("envelope.New: %v", err)
	}
	if read {
		e.Meta["read_at"] = time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if err := mailbox.Append(mailbox.RecipientDir(base, recipient), e); err != nil {
		t.Fatalf("mailbox.Append: %v", err)
	}
}

func TestNaclsListsMultipleMailboxes(t *testing.T) {
	base := t.TempDir()
	writeEnvelope(t, base, "zoe", "ops", false)
	writeEnvelope(t, base, "alice", "ops", false)
	writeEnvelope(t, base, "alice", "ops", true)

	out, err := runNaclsCmd(t, "--path", base)
	if err != nil {
		t.Fatalf("nacls: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header plus 2 mailbox rows, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[1], "alice") || !strings.Contains(lines[2], "zoe") {
		t.Fatalf("expected sorted recipients, got:\n%s", out)
	}
	if !strings.Contains(lines[1], "2") || !strings.Contains(lines[1], "1") {
		t.Fatalf("expected alice counts in output:\n%s", out)
	}
}

func TestNaclsRecipientJSONIncludesMalformedCount(t *testing.T) {
	base := t.TempDir()
	writeEnvelope(t, base, "alice", "ops", false)
	writeEnvelope(t, base, "alice", "ops", true)

	inboxPath := filepath.Join(base, "alice", "inbox")
	f, err := os.OpenFile(inboxPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("open inbox: %v", err)
	}
	if _, err := f.WriteString("{not-json}\n"); err != nil {
		t.Fatalf("write malformed line: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close inbox: %v", err)
	}

	out, err := runNaclsCmd(t, "--path", base, "--json", "alice")
	if err != nil {
		t.Fatalf("nacls --json alice: %v", err)
	}

	var summary map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &summary); err != nil {
		t.Fatalf("parse json: %v\nout: %s", err, out)
	}

	wantKeys := []string{"recipient", "total", "pending", "read", "malformed"}
	for _, key := range wantKeys {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing key %q in %v", key, summary)
		}
	}
	if summary["recipient"] != "alice" {
		t.Fatalf("recipient: got %v", summary["recipient"])
	}
	if int(summary["total"].(float64)) != 3 {
		t.Fatalf("total: got %v", summary["total"])
	}
	if int(summary["pending"].(float64)) != 1 {
		t.Fatalf("pending: got %v", summary["pending"])
	}
	if int(summary["read"].(float64)) != 1 {
		t.Fatalf("read: got %v", summary["read"])
	}
	if int(summary["malformed"].(float64)) != 1 {
		t.Fatalf("malformed: got %v", summary["malformed"])
	}
}

func TestNaclsJSONAllMailboxesSorted(t *testing.T) {
	base := t.TempDir()
	writeEnvelope(t, base, "zoe", "ops", false)
	writeEnvelope(t, base, "alice", "ops", false)

	out, err := runNaclsCmd(t, "--path", base, "--json")
	if err != nil {
		t.Fatalf("nacls --json: %v", err)
	}

	var summaries []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &summaries); err != nil {
		t.Fatalf("parse json: %v\nout: %s", err, out)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if summaries[0]["recipient"] != "alice" || summaries[1]["recipient"] != "zoe" {
		t.Fatalf("expected sorted recipients, got %v", summaries)
	}
}

func TestNaclsMissingPathExitsCleanly(t *testing.T) {
	base := filepath.Join(t.TempDir(), "missing")

	out, err := runNaclsCmd(t, "--path", base)
	if err != nil {
		t.Fatalf("nacls missing path: %v", err)
	}
	if strings.TrimSpace(out) != "RECIPIENT  TOTAL  PENDING  READ  MALFORMED" {
		t.Fatalf("unexpected missing-path output:\n%s", out)
	}

	out, err = runNaclsCmd(t, "--path", base, "--json")
	if err != nil {
		t.Fatalf("nacls missing path --json: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("expected empty JSON array, got %q", strings.TrimSpace(out))
	}
}
