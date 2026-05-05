package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/format"
	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
)

func newCostCmd() *cobra.Command {
	var window string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show estimated API-equivalent cost from local jsonl transcripts",
		Long: "Aggregates assistant token usage from ~/.claude/projects/**/*.jsonl and applies\n" +
			"public list pricing for Opus/Sonnet/Haiku. Machine-wide, not per-account.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs, err := jsonlscan.ScanCosts()
			if err != nil {
				return err
			}
			if asJSON {
				enc := jsonEncoder(cmd)
				switch window {
				case "today":
					return enc.Encode(cs.Today)
				case "7d":
					return enc.Encode(cs.Last7)
				case "30d":
					return enc.Encode(cs.Last30)
				default:
					return enc.Encode(cs)
				}
			}
			switch window {
			case "today":
				printReport(cmd, cs.Today, true)
			case "7d":
				printReport(cmd, cs.Last7, false)
			case "30d":
				printReport(cmd, cs.Last30, false)
				printDaily(cmd, cs.Last30.DailyTotals)
			default:
				printReport(cmd, cs.Today, true)
				fmt.Fprintln(cmd.OutOrStdout())
				printOneLine(cmd, "Last 7 days ", cs.Last7)
				printOneLine(cmd, "Last 30 days", cs.Last30)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "Note: prices are public list rates; cache-creation/read multipliers applied. Not an invoice.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&window, "window", "w", "", "today|7d|30d (default: summary of all)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func printReport(cmd *cobra.Command, r jsonlscan.CostReport, withActivity bool) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s — $%.2f  (%d turns, %s tokens)\n",
		strings.ToUpper(r.Window), r.Cost, r.Turns, format.HumanTokens(r.Tokens.Total()))
	if withActivity {
		fmt.Fprintf(out, "Sessions: %d   Active: %s",
			r.Sessions, format.HumanDuration(r.ActiveDur))
		if !r.LastActive.IsZero() {
			fmt.Fprintf(out, "   Last active: %s", r.LastActive.Format("15:04:05"))
		}
		fmt.Fprintln(out)
	}
	if len(r.ByFamily) == 0 {
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  FAMILY\tTURNS\tTOKENS\tCOST")
	for _, b := range r.ByFamily {
		fmt.Fprintf(w, "  %s\t%d\t%s\t$%.2f\n", b.Family, b.Turns, format.HumanTokens(b.Tokens.Total()), b.Cost)
	}
	w.Flush()
}

func printOneLine(cmd *cobra.Command, label string, r jsonlscan.CostReport) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s: $%-9.2f (%d turns, %s tokens)\n",
		label, r.Cost, r.Turns, format.HumanTokens(r.Tokens.Total()))
}

func printDaily(cmd *cobra.Command, daily []jsonlscan.DailyTotal) {
	if len(daily) == 0 {
		return
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Daily history:")
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  DATE\tTURNS\tCOST\tMODELS")
	for _, d := range daily {
		fmt.Fprintf(w, "  %s\t%d\t$%.2f\t%s\n", d.Date, d.Turns, d.Cost, strings.Join(d.Families, ", "))
	}
	w.Flush()
}

