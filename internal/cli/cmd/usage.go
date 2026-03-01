package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	cliconfig "github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "View current and historical usage",
	Long:  "Display your API usage for the current billing period",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cliconfig.NewManager()
		if err != nil {
			return fmt.Errorf("initializing config: %w", err)
		}

		token, err := mgr.Get(tokenKey)
		if err != nil || token == "" {
			return fmt.Errorf("not authenticated. Run 'omdr auth login' first")
		}

		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "https://cli.omdr.dev"
		}
		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		var usage struct {
			Period struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"period"`
			Requests   int     `json:"requests"`
			Compute    float64 `json:"compute_seconds"`
			Bandwidth  float64 `json:"bandwidth_mb"`
			CreditUsed float64 `json:"credit_used"`
			RateLimit  int     `json:"rate_limit"`
			RateUsed   int     `json:"rate_used"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/usage", &usage); err != nil {
			return fmt.Errorf("fetching usage: %w", err)
		}

		fmt.Printf("Billing Period: %s to %s\n\n", usage.Period.Start, usage.Period.End)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "METRIC\tVALUE")
		fmt.Fprintf(w, "API Requests\t%d\n", usage.Requests)
		fmt.Fprintf(w, "Compute\t%.1f seconds\n", usage.Compute)
		fmt.Fprintf(w, "Bandwidth\t%.1f MB\n", usage.Bandwidth)
		fmt.Fprintf(w, "Credits Used\t%.2f\n", usage.CreditUsed)
		if usage.RateLimit > 0 {
			fmt.Fprintf(w, "Rate Limit\t%d / %d req/min\n", usage.RateUsed, usage.RateLimit)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
