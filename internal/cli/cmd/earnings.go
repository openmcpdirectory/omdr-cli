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

var earningsCmd = &cobra.Command{
	Use:   "earnings",
	Short: "View publisher earnings and payouts",
	Long:  "Display your earnings summary, records, and payout history as a server publisher",
}

var earningsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "View earnings summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := authedClient()
		if err != nil {
			return err
		}

		var summary struct {
			TotalEarned    float64 `json:"total_earned"`
			TotalPaid      float64 `json:"total_paid"`
			PendingBalance float64 `json:"pending_balance"`
			Currency       string  `json:"currency"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/earnings/summary", &summary); err != nil {
			return fmt.Errorf("fetching earnings: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "Total Earned\t%.2f %s\n", summary.TotalEarned, summary.Currency)
		fmt.Fprintf(w, "Total Paid\t%.2f %s\n", summary.TotalPaid, summary.Currency)
		fmt.Fprintf(w, "Pending\t%.2f %s\n", summary.PendingBalance, summary.Currency)
		w.Flush()

		return nil
	},
}

var earningsPayoutsCmd = &cobra.Command{
	Use:   "payouts",
	Short: "View payout history",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := authedClient()
		if err != nil {
			return err
		}

		var payouts struct {
			Items []struct {
				ID        string  `json:"id"`
				Amount    float64 `json:"amount"`
				Currency  string  `json:"currency"`
				Status    string  `json:"status"`
				CreatedAt string  `json:"created_at"`
			} `json:"items"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/earnings/payouts", &payouts); err != nil {
			return fmt.Errorf("fetching payouts: %w", err)
		}

		if len(payouts.Items) == 0 {
			fmt.Println("No payouts yet.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "DATE\tAMOUNT\tSTATUS")
		for _, p := range payouts.Items {
			fmt.Fprintf(w, "%s\t%.2f %s\t%s\n", p.CreatedAt, p.Amount, p.Currency, p.Status)
		}
		w.Flush()

		return nil
	},
}

func authedClient() (*client.Client, error) {
	mgr, err := cliconfig.NewManager()
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	token, err := mgr.Get(tokenKey)
	if err != nil || token == "" {
		return nil, fmt.Errorf("not authenticated. Run 'omdr auth login' first")
	}

	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://cli.omdr.dev"
	}
	c := client.NewClient(apiURL)
	c.SetToken(token)
	return c, nil
}

func init() {
	rootCmd.AddCommand(earningsCmd)
	earningsCmd.AddCommand(earningsSummaryCmd)
	earningsCmd.AddCommand(earningsPayoutsCmd)
}
