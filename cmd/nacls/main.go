package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jylhis/nacutils/internal/envelope"
	"github.com/jylhis/nacutils/internal/mailbox"
)

type mailboxSummary struct {
	Recipient string `json:"recipient"`
	Total     int    `json:"total"`
	Pending   int    `json:"pending"`
	Read      int    `json:"read"`
	Malformed int    `json:"malformed"`
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		asJSON bool
		base   string
	)

	cmd := &cobra.Command{
		Use:           "nacls [recipient]",
		Short:         "list mailbox counts without opening envelopes",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := base
			if path == "" {
				path = mailbox.BaseDir()
			}

			out := cmd.OutOrStdout()
			if len(args) == 1 {
				summary, err := summarizeRecipient(path, args[0])
				if err != nil {
					return fmt.Errorf("nacls: %w", err)
				}
				return writeSummaries(out, []mailboxSummary{summary}, asJSON, true)
			}

			summaries, err := summarizeAll(path)
			if err != nil {
				return fmt.Errorf("nacls: %w", err)
			}
			return writeSummaries(out, summaries, asJSON, false)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON summaries")
	cmd.Flags().StringVar(&base, "path", "", "override mail root path (default: nacutils mail dir)")
	return cmd
}

func writeSummaries(w io.Writer, summaries []mailboxSummary, asJSON bool, single bool) error {
	if asJSON {
		encoder := json.NewEncoder(w)
		if single {
			return encoder.Encode(summaries[0])
		}
		if summaries == nil {
			summaries = []mailboxSummary{}
		}
		return encoder.Encode(summaries)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "RECIPIENT\tTOTAL\tPENDING\tREAD\tMALFORMED")
	for _, summary := range summaries {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\n",
			summary.Recipient,
			summary.Total,
			summary.Pending,
			summary.Read,
			summary.Malformed,
		)
	}
	return tw.Flush()
}

func summarizeAll(base string) ([]mailboxSummary, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var summaries []mailboxSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary, err := summarizeRecipient(base, entry.Name())
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Recipient < summaries[j].Recipient
	})
	return summaries, nil
}

func summarizeRecipient(base, recipient string) (mailboxSummary, error) {
	summary := mailboxSummary{Recipient: recipient}

	f, err := os.Open(mailbox.InboxPath(mailbox.RecipientDir(base, recipient)))
	if err != nil {
		if os.IsNotExist(err) {
			return summary, nil
		}
		return mailboxSummary{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		summary.Total++
		e, err := envelope.Unmarshal(line)
		if err != nil {
			summary.Malformed++
			continue
		}
		if isReadEnvelope(e) {
			summary.Read++
			continue
		}
		summary.Pending++
	}
	if err := scanner.Err(); err != nil {
		return mailboxSummary{}, err
	}

	return summary, nil
}

func isReadEnvelope(e *envelope.Envelope) bool {
	if e == nil || e.Meta == nil {
		return false
	}
	_, ok := e.Meta["read_at"]
	return ok
}
