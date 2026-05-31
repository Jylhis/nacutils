package envelope_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jylhis/nacutils/internal/envelope"
)

func TestNew(t *testing.T) {
	e, err := envelope.New("alice", "bob", envelope.KindNote, "Hello", "body text")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.Sender != "alice" {
		t.Errorf("sender: got %q, want %q", e.Sender, "alice")
	}
	if e.Recipient != "bob" {
		t.Errorf("recipient: got %q, want %q", e.Recipient, "bob")
	}
	if e.Kind != envelope.KindNote {
		t.Errorf("kind: got %q, want %q", e.Kind, envelope.KindNote)
	}
	if e.Subject != "Hello" {
		t.Errorf("subject: got %q, want %q", e.Subject, "Hello")
	}
	if e.Body != "body text" {
		t.Errorf("body: got %q, want %q", e.Body, "body text")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if e.CreatedAt.Location() != time.UTC {
		t.Error("expected UTC created_at")
	}
	if e.Meta == nil {
		t.Error("expected non-nil meta")
	}
}

func TestNewWithID(t *testing.T) {
	id := "0195fba5-6c11-7000-8000-000000000001"
	e, err := envelope.NewWithID(id, "a", "b", envelope.KindStatus, "", "body")
	if err != nil {
		t.Fatalf("NewWithID: %v", err)
	}
	if e.ID != id {
		t.Errorf("id: got %q, want %q", e.ID, id)
	}
}

func TestNewWithID_Invalid(t *testing.T) {
	_, err := envelope.NewWithID("not-a-uuid", "a", "b", envelope.KindNote, "", "body")
	if err == nil {
		t.Error("expected error for invalid id")
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	e, err := envelope.New("sender", "recipient", envelope.KindAttn, "subj", "body")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := envelope.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := envelope.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != e.ID {
		t.Errorf("roundtrip id: got %q, want %q", got.ID, e.ID)
	}
	if got.Body != e.Body {
		t.Errorf("roundtrip body: got %q, want %q", got.Body, e.Body)
	}
	if !got.CreatedAt.Equal(e.CreatedAt) {
		t.Errorf("roundtrip created_at: got %v, want %v", got.CreatedAt, e.CreatedAt)
	}
}

func TestMarshalSchema(t *testing.T) {
	e, err := envelope.New("s", "r", envelope.KindHeartbeatSummary, "", "hello")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data, err := envelope.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	requiredFields := []string{"id", "sender", "recipient", "kind", "body", "created_at", "meta"}
	for _, f := range requiredFields {
		if _, ok := m[f]; !ok {
			t.Errorf("missing field %q in marshaled envelope", f)
		}
	}
}

func TestParseKind(t *testing.T) {
	cases := []struct {
		input   string
		want    envelope.Kind
		wantErr bool
	}{
		{"note", envelope.KindNote, false},
		{"status", envelope.KindStatus, false},
		{"attn", envelope.KindAttn, false},
		{"heartbeat-summary", envelope.KindHeartbeatSummary, false},
		{"invalid", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		got, err := envelope.ParseKind(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseKind(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseKind(%q): %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseKind(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestUnmarshal_NilMeta(t *testing.T) {
	raw := `{"id":"x","sender":"a","recipient":"b","kind":"note","body":"hi","created_at":"2026-01-01T00:00:00Z","meta":null}`
	e, err := envelope.Unmarshal([]byte(raw))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if e.Meta == nil {
		t.Error("expected non-nil meta after Unmarshal with null meta")
	}
	_ = strings.Contains("unused", "import")
}
