package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jylhis/nacutils/internal/cliutil"
	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nacwrite",
		Short: "non-interactive nacmail envelope authoring",
		Long: `nacwrite creates nacmail-compatible envelopes without any TTY dependency.

Use --body for fully flag-driven sends, or pipe body content on stdin when --body is omitted.`,
		SilenceUsage: true,
	}

	root.AddCommand(newSendCmd())
	return root
}

func newSendCmd() *cobra.Command {
	var (
		kind    string
		subject string
		sender  string
		id      string
		body    string
		metaRaw string
		asJSON  bool
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "send <recipient>",
		Short: "write a nacmail envelope non-interactively",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipient := args[0]

			k, err := envelope.ParseKind(kind)
			if err != nil {
				return err
			}

			resolvedBody, err := resolveBody(cmd, body)
			if err != nil {
				return err
			}

			meta, err := parseMeta(metaRaw)
			if err != nil {
				return err
			}

			var e *envelope.Envelope
			if id != "" {
				e, err = envelope.NewWithID(id, sender, recipient, k, subject, resolvedBody)
			} else {
				e, err = envelope.New(sender, recipient, k, subject, resolvedBody)
			}
			if err != nil {
				return fmt.Errorf("create envelope: %w", err)
			}
			e.Meta = meta

			if !dryRun {
				base := mailbox.BaseDir()
				dir := mailbox.RecipientDir(base, recipient)
				if err := mailbox.Append(dir, e); err != nil {
					return fmt.Errorf("send: %w", err)
				}
			}

			out := cmd.OutOrStdout()
			if asJSON {
				data, err := envelope.Marshal(e)
				if err != nil {
					return fmt.Errorf("marshal envelope: %w", err)
				}
				_, err = fmt.Fprintln(out, string(data))
				return err
			}

			_, err = fmt.Fprintln(out, e.ID)
			return err
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "note", "message kind (note|status|attn|heartbeat-summary)")
	cmd.Flags().StringVar(&subject, "subject", "", "message subject")
	cmd.Flags().StringVar(&sender, "sender", cliutil.CurrentUsername(), "sender identity (default: current user)")
	cmd.Flags().StringVar(&id, "id", "", "envelope ID (UUIDv7); auto-generated if not set")
	cmd.Flags().StringVar(&body, "body", "", "message body; reads from stdin when omitted")
	cmd.Flags().StringVar(&metaRaw, "meta", "", "metadata JSON object")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output the full envelope as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and render output without writing to the mailbox")

	return cmd
}

func resolveBody(cmd *cobra.Command, body string) (string, error) {
	if body != "" {
		return body, nil
	}

	in := cmd.InOrStdin()
	if cliutil.IsTTY(in) {
		return "", fmt.Errorf("body is required via --body or stdin")
	}

	data, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("body is required via --body or stdin")
	}

	return string(data), nil
}

func parseMeta(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, fmt.Errorf("parse --meta: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("--meta must be a JSON object")
	}
	return meta, nil
}
