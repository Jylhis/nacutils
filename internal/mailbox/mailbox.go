package mailbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	inboxPath := InboxPath(recipientDir)
	f, err := os.OpenFile(inboxPath, os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range kept {
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
