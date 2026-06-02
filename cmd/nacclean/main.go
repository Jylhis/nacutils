package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/jylhis/nacutils/internal/mailbox"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		beforeRaw string
		apply     bool
		asJSON    bool
	)

	cmd := &cobra.Command{
		Use:   "nacclean [recipient]",
		Short: "remove read envelopes older than a threshold",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := parseBefore(beforeRaw, time.Now().UTC())
			if err != nil {
				return err
			}

			recipient := ""
			if len(args) == 1 {
				recipient = args[0]
			}

			report, err := mailbox.Cleanup(mailbox.BaseDir(), recipient, before, apply)
			if err != nil {
				return fmt.Errorf("nacclean: %w", err)
			}

			out := cmd.OutOrStdout()
			if asJSON {
				payload := map[string]any{
					"recipient":          report.RecipientScope,
					"matched_recipients": report.MatchedRecipients,
					"inspected":          report.Inspected,
					"eligible":           report.Eligible,
					"removed":            report.Removed,
					"dry_run":            !report.Apply,
					"before":             report.Before.Format(time.RFC3339),
				}
				data, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				fmt.Fprintln(out, string(data))
				return nil
			}

			scope := report.RecipientScope
			if scope == "" {
				scope = "*"
			}
			fmt.Fprintf(
				out,
				"recipient=%s matched_recipients=%d inspected=%d eligible=%d removed=%d dry_run=%t before=%s\n",
				scope,
				report.MatchedRecipients,
				report.Inspected,
				report.Eligible,
				report.Removed,
				!report.Apply,
				report.Before.Format(time.RFC3339),
			)
			return nil
		},
	}

	cmd.Flags().StringVar(&beforeRaw, "before", "", "delete mail created before this duration, RFC3339 timestamp, or unix-ms timestamp")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply deletions; default is dry-run")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output cleanup report as JSON")
	_ = cmd.MarkFlagRequired("before")
	return cmd
}

func parseBefore(raw string, now time.Time) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("required flag(s) \"before\" not set")
	}

	if d, err := time.ParseDuration(raw); err == nil {
		return now.Add(-d).UTC(), nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC(), nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid --before %q: must be duration, RFC3339, or unix-ms", raw)
}
