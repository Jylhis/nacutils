package mailbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jylhis/nacutils/internal/envelope"
)

// BaseDir returns the root mail directory, respecting XDG_DATA_HOME.
func BaseDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "nacutils", "mail")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nacutils", "mail")
}

// RecipientDir returns the mailbox directory for a given recipient.
func RecipientDir(base, recipient string) string {
	return filepath.Join(base, recipient)
}

// InboxPath returns the inbox file path for a recipient directory.
func InboxPath(recipientDir string) string {
	return filepath.Join(recipientDir, "inbox")
}

// Append writes an envelope to the recipient's inbox.
// If an envelope with the same ID already exists, the call is a no-op (idempotent).
func Append(recipientDir string, e *envelope.Envelope) error {
	if err := os.MkdirAll(recipientDir, 0700); err != nil {
		return fmt.Errorf("create mailbox dir: %w", err)
	}

	existing, err := ReadAll(recipientDir)
	if err != nil {
		return fmt.Errorf("read inbox: %w", err)
	}
	for _, ex := range existing {
		if ex.ID == e.ID {
			return nil
		}
	}

	f, err := os.OpenFile(InboxPath(recipientDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open inbox: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// ReadAll reads all envelopes from a recipient's inbox file.
// Returns nil slice (not an error) when the inbox does not exist.
func ReadAll(recipientDir string) ([]*envelope.Envelope, error) {
	f, err := os.Open(InboxPath(recipientDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var envelopes []*envelope.Envelope
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		e, err := envelope.Unmarshal(line)
		if err != nil {
			return nil, fmt.Errorf("parse envelope: %w", err)
		}
		envelopes = append(envelopes, e)
	}
	return envelopes, scanner.Err()
}

// FindByID searches all recipient inboxes under baseDir for an envelope with the given ID.
// Returns the envelope and the recipientDir it was found in.
func FindByID(baseDir, id string) (*envelope.Envelope, string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("envelope %s not found", id)
		}
		return nil, "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(baseDir, entry.Name())
		envelopes, err := ReadAll(dir)
		if err != nil {
			continue
		}
		for _, e := range envelopes {
			if e.ID == id {
				return e, dir, nil
			}
		}
	}
	return nil, "", fmt.Errorf("envelope %s not found", id)
}

// Remove deletes an envelope by ID from its recipient's inbox.
// The inbox file is rewritten without the matching line.
func Remove(recipientDir, id string) error {
	envelopes, err := ReadAll(recipientDir)
	if err != nil {
		return err
	}

	var kept []*envelope.Envelope
	found := false
	for _, e := range envelopes {
		if e.ID == id {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return fmt.Errorf("envelope %s not found", id)
	}

	return WriteAll(recipientDir, kept)
}

// WriteAll rewrites a recipient inbox with the provided envelopes.
func WriteAll(recipientDir string, envelopes []*envelope.Envelope) error {
	if err := os.MkdirAll(recipientDir, 0700); err != nil {
		return fmt.Errorf("create mailbox dir: %w", err)
	}

	f, err := os.OpenFile(InboxPath(recipientDir), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range envelopes {
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
			return err
		}
	}
	return nil
}

// MarkRead records read metadata for an envelope the first time it is read.
func MarkRead(baseDir, id string, readAt time.Time) error {
	_, recipientDir, err := FindByID(baseDir, id)
	if err != nil {
		return err
	}

	envelopes, err := ReadAll(recipientDir)
	if err != nil {
		return err
	}

	for _, e := range envelopes {
		if e.ID != id {
			continue
		}
		if e.Meta == nil {
			e.Meta = map[string]any{}
		}
		if _, ok := e.Meta["read_at"]; !ok {
			e.Meta["read_at"] = readAt.UTC().Format(time.RFC3339)
		}
		return WriteAll(recipientDir, envelopes)
	}

	return fmt.Errorf("envelope %s not found", id)
}

type CleanupReport struct {
	RecipientScope    string
	MatchedRecipients int
	Inspected         int
	Eligible          int
	Removed           int
	Before            time.Time
	Apply             bool
}

// Cleanup deletes read envelopes older than before. Without apply, it only reports.
func Cleanup(baseDir string, recipient string, before time.Time, apply bool) (CleanupReport, error) {
	report := CleanupReport{
		RecipientScope: recipient,
		Before:         before.UTC(),
		Apply:          apply,
	}

	recipients, err := cleanupRecipients(baseDir, recipient)
	if err != nil {
		return report, err
	}
	report.MatchedRecipients = len(recipients)

	for _, recipientDir := range recipients {
		envelopes, err := ReadAll(recipientDir)
		if err != nil {
			return report, err
		}

		var kept []*envelope.Envelope
		changed := false
		for _, e := range envelopes {
			report.Inspected++
			if cleanupEligible(e, before) {
				report.Eligible++
				if apply {
					report.Removed++
					changed = true
					continue
				}
			}
			kept = append(kept, e)
		}

		if apply && changed {
			if err := WriteAll(recipientDir, kept); err != nil {
				return report, err
			}
		}
	}

	return report, nil
}

func cleanupRecipients(baseDir, recipient string) ([]string, error) {
	if recipient != "" {
		dir := RecipientDir(baseDir, recipient)
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return []string{dir}, nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(baseDir, entry.Name()))
		}
	}
	return dirs, nil
}

func cleanupEligible(e *envelope.Envelope, before time.Time) bool {
	if !e.CreatedAt.Before(before) {
		return false
	}
	if e.Meta == nil {
		return false
	}
	_, ok := e.Meta["read_at"]
	return ok
}
