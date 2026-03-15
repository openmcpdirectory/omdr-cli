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
			TotalEarnings    int    `json:"totalEarnings"`
			PendingPayout    int    `json:"pendingPayout"`
			MonthlyEarnings  int    `json:"monthlyEarnings"`
			LastPayoutAmount int    `json:"lastPayoutAmount"`
			LastPayoutDate   string `json:"lastPayoutDate,omitempty"`
			PayoutConfigured bool   `json:"payoutConfigured"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/earnings/summary", &summary); err != nil {
			return fmt.Errorf("fetching earnings: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "Total Earnings\t%d credits\n", summary.TotalEarnings)
		fmt.Fprintf(w, "Pending Payout\t%d credits\n", summary.PendingPayout)
		fmt.Fprintf(w, "Monthly Earnings\t%d credits\n", summary.MonthlyEarnings)
		fmt.Fprintf(w, "Payout Configured\t%v\n", summary.PayoutConfigured)
		if summary.LastPayoutAmount > 0 {
			fmt.Fprintf(w, "Last Payout\t%d credits on %s\n", summary.LastPayoutAmount, summary.LastPayoutDate)
		}
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

		var payoutsResp struct {
			Payouts []struct {
				ID        string `json:"id"`
				Amount    int    `json:"amount"`
				Status    string `json:"status"`
				CreatedAt string `json:"createdAt"`
				PaidAt    string `json:"paidAt,omitempty"`
			} `json:"payouts"`
			Total int `json:"total"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/earnings/payouts", &payoutsResp); err != nil {
			return fmt.Errorf("fetching payouts: %w", err)
		}

		if len(payoutsResp.Payouts) == 0 {
			fmt.Println("No payouts yet.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "DATE\tAMOUNT\tSTATUS")
		for _, p := range payoutsResp.Payouts {
			fmt.Fprintf(w, "%s\t%d credits\t%s\n", p.CreatedAt, p.Amount, p.Status)
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
		apiURL = defaults.CLIURL
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
