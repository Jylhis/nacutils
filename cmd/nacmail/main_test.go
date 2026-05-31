package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jylhis/nacutils/internal/envelope"
)

func runCmd(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf.String(), err
}

func TestCLI_SendListReadRm(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	// send
	out, err := runCmd(t, "send", "testuser", "hello world", "--sender", "testbot", "--kind", "note", "--subject", "greet")
	if err != nil {
		t.Fatalf("send: %v\nout: %s", err, out)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("send: expected non-empty id in output")
	}

	// list (table)
	out, err = runCmd(t, "list", "testuser")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("list output missing id %q:\n%s", id, out)
	}
	if !strings.Contains(out, "testbot") {
		t.Errorf("list output missing sender:\n%s", out)
	}

	// list --json
	out, err = runCmd(t, "list", "testuser", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var e envelope.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e); err != nil {
		t.Fatalf("list --json: not valid JSON: %v\nout: %s", err, out)
	}
	if e.ID != id {
		t.Errorf("list --json: id mismatch: got %q, want %q", e.ID, id)
	}

	// read
	out, err = runCmd(t, "read", id)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(out, id) {
		t.Errorf("read output missing id:\n%s", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("read output missing body:\n%s", out)
	}

	// read --json
	out, err = runCmd(t, "read", id, "--json")
	if err != nil {
		t.Fatalf("read --json: %v", err)
	}
	var e2 envelope.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e2); err != nil {
		t.Fatalf("read --json: not valid JSON: %v\nout: %s", err, out)
	}
	if e2.Body != "hello world" {
		t.Errorf("read --json: body mismatch: got %q", e2.Body)
	}

	// rm
	_, err = runCmd(t, "rm", id)
	if err != nil {
		t.Fatalf("rm: %v", err)
	}

	// read after rm → error
	_, err = runCmd(t, "read", id)
	if err == nil {
		t.Error("read after rm: expected error")
	}
}

func TestCLI_Send_Idempotent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fixedID := "019e8007-0000-7000-8000-000000000001"

	for i := 0; i < 3; i++ {
		out, err := runCmd(t, "send", "bob", "idempotent", "--id", fixedID)
		if err != nil {
			t.Fatalf("send attempt %d: %v", i, err)
		}
		if strings.TrimSpace(out) != fixedID {
			t.Errorf("send attempt %d: expected id %q, got %q", i, fixedID, strings.TrimSpace(out))
		}
	}

	out, err := runCmd(t, "list", "bob", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("idempotent send: expected 1 envelope, got %d", len(lines))
	}
}

func TestCLI_Send_InvalidKind(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := runCmd(t, "send", "alice", "body", "--kind", "bad-kind")
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestCLI_Read_NotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := runCmd(t, "read", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent envelope")
	}
}

func TestCLI_Rm_NotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := runCmd(t, "rm", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent envelope")
	}
}

func TestCLI_List_HeartbeatSummaryKind(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := runCmd(t, "send", "ceo", "daily summary", "--kind", "heartbeat-summary", "--sender", "agent-001")
	if err != nil {
		t.Fatalf("send heartbeat-summary: %v", err)
	}

	out, err := runCmd(t, "list", "ceo", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var e envelope.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e); err != nil {
		t.Fatalf("list --json parse: %v", err)
	}
	if e.Kind != "heartbeat-summary" {
		t.Errorf("kind: got %q, want %q", e.Kind, "heartbeat-summary")
	}
	if e.Sender != "agent-001" {
		t.Errorf("sender: got %q, want %q", e.Sender, "agent-001")
	}
}
