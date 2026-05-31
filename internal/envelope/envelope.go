package envelope

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindNote             Kind = "note"
	KindStatus           Kind = "status"
	KindAttn             Kind = "attn"
	KindHeartbeatSummary Kind = "heartbeat-summary"
)

var validKinds = map[Kind]bool{
	KindNote:             true,
	KindStatus:           true,
	KindAttn:             true,
	KindHeartbeatSummary: true,
}

func ParseKind(s string) (Kind, error) {
	k := Kind(s)
	if !validKinds[k] {
		return "", fmt.Errorf("invalid kind %q: must be note, status, attn, or heartbeat-summary", s)
	}
	return k, nil
}

// Envelope is the v1 message container shared across nacutils tools.
type Envelope struct {
	ID        string         `json:"id"`
	Sender    string         `json:"sender"`
	Recipient string         `json:"recipient"`
	Kind      Kind           `json:"kind"`
	Subject   string         `json:"subject,omitempty"`
	Body      string         `json:"body"`
	CreatedAt time.Time      `json:"created_at"`
	Meta      map[string]any `json:"meta"`
}

// New creates a new Envelope with a freshly generated UUIDv7 ID.
func New(sender, recipient string, kind Kind, subject, body string) (*Envelope, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	return &Envelope{
		ID:        id.String(),
		Sender:    sender,
		Recipient: recipient,
		Kind:      kind,
		Subject:   subject,
		Body:      body,
		CreatedAt: time.Now().UTC(),
		Meta:      map[string]any{},
	}, nil
}

// NewWithID creates a new Envelope with the given ID (must be a valid UUIDv7).
func NewWithID(id, sender, recipient string, kind Kind, subject, body string) (*Envelope, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid id %q: %w", id, err)
	}
	return &Envelope{
		ID:        id,
		Sender:    sender,
		Recipient: recipient,
		Kind:      kind,
		Subject:   subject,
		Body:      body,
		CreatedAt: time.Now().UTC(),
		Meta:      map[string]any{},
	}, nil
}

func Marshal(e *Envelope) ([]byte, error) {
	return json.Marshal(e)
}

func Unmarshal(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	if e.Meta == nil {
		e.Meta = map[string]any{}
	}
	return &e, nil
}
