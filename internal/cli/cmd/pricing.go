package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/defaults"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "Display pricing tiers",
	Long:  "Show available subscription tiers and their pricing",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = defaults.CLIURL
		}
		apiClient := client.NewClient(apiURL)

		var tiers struct {
			Items []struct {
				Name      string   `json:"name"`
				Price     float64  `json:"price"`
				Currency  string   `json:"currency"`
				Interval  string   `json:"interval"`
				Requests  int      `json:"requests_per_month"`
				Compute   int      `json:"compute_seconds"`
				Bandwidth int      `json:"bandwidth_mb"`
				Features  []string `json:"features"`
			} `json:"tiers"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/sponsorships/pricing", &tiers); err != nil {
			return fmt.Errorf("fetching pricing: %w", err)
		}

		if len(tiers.Items) == 0 {
			fmt.Println("No pricing tiers available.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "TIER\tPRICE\tREQUESTS\tCOMPUTE\tBANDWIDTH")
		for _, t := range tiers.Items {
			price := "Free"
			if t.Price > 0 {
				price = fmt.Sprintf("%.2f %s/%s", t.Price, t.Currency, t.Interval)
			}
			fmt.Fprintf(w, "%s\t%s\t%d/mo\t%ds\t%d MB\n",
				t.Name, price, t.Requests, t.Compute, t.Bandwidth)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pricingCmd)
}
