package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
)

func newPricingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Show or scaffold the pricing override file",
	}
	cmd.AddCommand(newPricingShowCmd(), newPricingInitCmd(), newPricingPathCmd())
	return cmd
}

func newPricingShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the resolved pricing table (overrides + defaults)",
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides, err := jsonlscan.LoadPricingOverrides()
			if err != nil {
				return err
			}
			out := make([]map[string]any, 0)
			seen := map[string]bool{}
			for _, e := range overrides {
				out = append(out, map[string]any{
					"source":  "override",
					"match":   e.Match,
					"family":  e.Family,
					"pricing": e.Pricing,
				})
				seen[e.Family] = true
			}
			for _, e := range jsonlscan.DefaultPricing {
				if seen[e.Family] {
					continue
				}
				out = append(out, map[string]any{
					"source":  "default",
					"match":   e.Match,
					"family":  e.Family,
					"pricing": e.Pricing,
				})
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
}

func newPricingInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter pricing.json the user can edit",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := jsonlscan.PricingFile()
			if force {
				_ = os.Remove(path)
			}
			p, err := jsonlscan.WriteDefaultPricingFile()
			if err != nil {
				return err
			}
			fmt.Printf("Pricing file: %s\n", p)
			fmt.Println("Edit it and re-run `ccm cost` to see the new numbers.")
			fmt.Println("Tip: to align with CCSwitcher's totals, set cache_create_5m_mult and cache_create_1h_mult to 0.1 (treats cache writes like cache reads).")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing pricing.json")
	return cmd
}

func newPricingPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the pricing file path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(jsonlscan.PricingFile())
		},
	}
}
