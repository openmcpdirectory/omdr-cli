package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	cliconfig "github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/defaults"
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
			apiURL = defaults.CLIURL
		}
		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		// API returns []UsageDataPoint: [{date, calls, cost}, ...]
		var dataPoints []struct {
			Date  string `json:"date"`
			Calls int    `json:"calls"`
			Cost  int    `json:"cost"` // in cents
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/usage", &dataPoints); err != nil {
			return fmt.Errorf("fetching usage: %w", err)
		}

		if len(dataPoints) == 0 {
			fmt.Println("No usage data found for the current period.")
			return nil
		}

		// Aggregate totals
		totalCalls := 0
		totalCost := 0
		for _, dp := range dataPoints {
			totalCalls += dp.Calls
			totalCost += dp.Cost
		}

		fmt.Printf("Usage (%d days):\n\n", len(dataPoints))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "DATE\tCALLS\tCOST")
		for _, dp := range dataPoints {
			fmt.Fprintf(w, "%s\t%d\t$%.2f\n", dp.Date, dp.Calls, float64(dp.Cost)/100)
		}
		fmt.Fprintln(w, "---\t---\t---")
		fmt.Fprintf(w, "Total\t%d\t$%.2f\n", totalCalls, float64(totalCost)/100)
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
