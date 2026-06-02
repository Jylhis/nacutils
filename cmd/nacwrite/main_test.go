package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

func runCmd(t *testing.T, input string, args ...string) (stdout string, err error) {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(input))
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), err
}

func TestSendWritesEnvelopeFromBodyFlag(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	out, err := runCmd(t, "", "send", "ceo", "--kind", "status", "--subject", "sync", "--body", "all clear", "--meta", `{"ticket":"JYL-61"}`, "--sender", "FoundingEngineer", "--json")
	if err != nil {
		t.Fatalf("send: %v\nout: %s", err, out)
	}

	var e envelope.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e); err != nil {
		t.Fatalf("json output: %v\nout: %s", err, out)
	}
	if e.Kind != envelope.KindStatus {
		t.Fatalf("kind: got %q", e.Kind)
	}
	if e.Meta["ticket"] != "JYL-61" {
		t.Fatalf("meta ticket: got %#v", e.Meta["ticket"])
	}

	envelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "ceo"))
	if err != nil {
		t.Fatalf("read mailbox: %v", err)
	}
	if len(envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envelopes))
	}
	if envelopes[0].Body != "all clear" {
		t.Fatalf("body: got %q", envelopes[0].Body)
	}
}

func TestSendReadsBodyFromStdin(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	out, err := runCmd(t, "body from stdin", "send", "ceo")
	if err != nil {
		t.Fatalf("send: %v\nout: %s", err, out)
	}

	id := strings.TrimSpace(out)
	envelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "ceo"))
	if err != nil {
		t.Fatalf("read mailbox: %v", err)
	}
	if len(envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(envelopes))
	}
	if envelopes[0].ID != id {
		t.Fatalf("id mismatch: got %q want %q", envelopes[0].ID, id)
	}
	if envelopes[0].Body != "body from stdin" {
		t.Fatalf("body mismatch: got %q", envelopes[0].Body)
	}
}

func TestSendMissingRecipient(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, err := runCmd(t, "body", "send")
	if err == nil {
		t.Fatal("expected missing recipient error")
	}
}

func TestSendInvalidKind(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, err := runCmd(t, "body", "send", "ceo", "--kind", "bad-kind")
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func TestSendInvalidMeta(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, err := runCmd(t, "", "send", "ceo", "--body", "body", "--meta", "{bad")
	if err == nil {
		t.Fatal("expected invalid meta error")
	}
}

func TestSendDryRunDoesNotWrite(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	out, err := runCmd(t, "body from stdin", "send", "ceo", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("dry-run: %v\nout: %s", err, out)
	}

	var e envelope.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e); err != nil {
		t.Fatalf("json output: %v\nout: %s", err, out)
	}
	if e.Body != "body from stdin" {
		t.Fatalf("body mismatch: got %q", e.Body)
	}

	envelopes, err := mailbox.ReadAll(mailbox.RecipientDir(mailbox.BaseDir(), "ceo"))
	if err != nil {
		t.Fatalf("read mailbox: %v", err)
	}
	if len(envelopes) != 0 {
		t.Fatalf("dry-run wrote %d envelopes", len(envelopes))
	}
}
