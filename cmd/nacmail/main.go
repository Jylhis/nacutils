package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
	"github.com/jylhis/nacutils/internal/render"
)

var writerIsTTY = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nacmail",
		Short: "async per-recipient message delivery for agents and users",
		Long: `nacmail is a local async mailbox for agent-to-agent and agent-to-user messaging.

Messages are stored as JSON-lines under $XDG_DATA_HOME/nacutils/mail/<recipient>/inbox.`,
		SilenceUsage: true,
	}

	root.AddCommand(newSendCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newReadCmd())
	root.AddCommand(newRmCmd())
	return root
}

func currentUsername() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}

// newSendCmd implements: nacmail send <recipient> <body>
func newSendCmd() *cobra.Command {
	var (
		kind    string
		subject string
		sender  string
		id      string
	)

	cmd := &cobra.Command{
		Use:   "send <recipient> <body>",
		Short: "send a message to a recipient's mailbox",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipient := args[0]
			body := args[1]

			k, err := envelope.ParseKind(kind)
			if err != nil {
				return err
			}

			var e *envelope.Envelope
			if id != "" {
				e, err = envelope.NewWithID(id, sender, recipient, k, subject, body)
			} else {
				e, err = envelope.New(sender, recipient, k, subject, body)
			}
			if err != nil {
				return fmt.Errorf("create envelope: %w", err)
			}

			base := mailbox.BaseDir()
			dir := mailbox.RecipientDir(base, recipient)
			if err := mailbox.Append(dir, e); err != nil {
				return fmt.Errorf("send: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), e.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "note", "message kind (note|status|attn|heartbeat-summary)")
	cmd.Flags().StringVar(&subject, "subject", "", "message subject")
	cmd.Flags().StringVar(&sender, "sender", currentUsername(), "sender identity (default: current user)")
	cmd.Flags().StringVar(&id, "id", "", "envelope ID (UUIDv7); auto-generated if not set")

	return cmd
}

// newListCmd implements: nacmail list [<recipient>]
func newListCmd() *cobra.Command {
	var asJSON bool
	var forceColor bool
	var noColor bool

	cmd := &cobra.Command{
		Use:   "list [<recipient>]",
		Short: "list envelopes in a recipient's mailbox (default: current user)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipient := currentUsername()
			if len(args) == 1 {
				recipient = args[0]
			}

			base := mailbox.BaseDir()
			dir := mailbox.RecipientDir(base, recipient)
			envelopes, err := mailbox.ReadAll(dir)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			out := cmd.OutOrStdout()

			if asJSON {
				for _, e := range envelopes {
					data, err := envelope.Marshal(e)
					if err != nil {
						return err
					}
					fmt.Fprintln(out, string(data))
				}
				return nil
			}

			if len(envelopes) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "no mail for %s\n", recipient)
				return nil
			}

			styled := shouldUseANSI(out, asJSON, forceColor, noColor)
			renderer := render.New()
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				renderer.Header("ID", styled),
				renderer.Header("FROM", styled),
				renderer.Header("KIND", styled),
				renderer.Header("SUBJECT", styled),
				renderer.Header("DATE", styled),
			)
			for _, e := range envelopes {
				subj := e.Subject
				if subj == "" {
					subj = "-"
				}
				date := e.CreatedAt.UTC().Format(time.DateTime)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ID, e.Sender, renderer.Kind(e.Kind, styled), subj, date)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output raw JSON (one envelope per line)")
	cmd.Flags().BoolVar(&forceColor, "color", false, "force ANSI styling when not using JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable ANSI styling")
	return cmd
}

// newReadCmd implements: nacmail read <id>
func newReadCmd() *cobra.Command {
	var asJSON bool
	var forceColor bool
	var noColor bool

	cmd := &cobra.Command{
		Use:   "read <id>",
		Short: "print a single envelope by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			base := mailbox.BaseDir()

			e, _, err := mailbox.FindByID(base, id)
			if err != nil {
				return err
			}
			readAt := time.Now().UTC()
			if err := mailbox.MarkRead(base, id, readAt); err != nil {
				return fmt.Errorf("read: %w", err)
			}
			if e.Meta == nil {
				e.Meta = map[string]any{}
			}
			if _, ok := e.Meta["read_at"]; !ok {
				e.Meta["read_at"] = readAt.Format(time.RFC3339)
			}

			out := cmd.OutOrStdout()
			if asJSON {
				data, err := envelope.Marshal(e)
				if err != nil {
					return err
				}
				fmt.Fprintln(out, string(data))
				return nil
			}

			printEnvelope(out, e, shouldUseANSI(out, asJSON, forceColor, noColor))
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output raw JSON")
	cmd.Flags().BoolVar(&forceColor, "color", false, "force ANSI styling when not using JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable ANSI styling")
	return cmd
}

// newRmCmd implements: nacmail rm <id>
func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "delete an envelope from the mailbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			base := mailbox.BaseDir()

			_, dir, err := mailbox.FindByID(base, id)
			if err != nil {
				return err
			}

			if err := mailbox.Remove(dir, id); err != nil {
				return fmt.Errorf("rm: %w", err)
			}
			return nil
		},
	}
}

func printEnvelope(w io.Writer, e *envelope.Envelope, styled bool) {
	renderer := render.New()

	fmt.Fprintf(w, "ID:        %s\n", e.ID)
	fmt.Fprintf(w, "From:      %s\n", e.Sender)
	fmt.Fprintf(w, "To:        %s\n", e.Recipient)
	fmt.Fprintf(w, "Kind:      %s\n", renderer.Kind(e.Kind, styled))
	if e.Subject != "" {
		fmt.Fprintf(w, "Subject:   %s\n", e.Subject)
	}
	fmt.Fprintf(w, "Date:      %s\n", e.CreatedAt.UTC().Format(time.RFC3339))
	if len(e.Meta) > 0 {
		data, _ := json.Marshal(e.Meta)
		fmt.Fprintf(w, "Meta:      %s\n", data)
	}
	fmt.Fprintln(w, renderer.Separator(strings.Repeat("-", 40), styled))
	fmt.Fprintln(w, renderer.KindText(e.Kind, e.Body, styled))
}

func shouldUseANSI(w io.Writer, asJSON bool, forceColor bool, noColor bool) bool {
	if asJSON || noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	if forceColor {
		return true
	}
	return writerIsTTY(w)
}
