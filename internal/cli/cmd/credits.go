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

var creditsCmd = &cobra.Command{
	Use:   "credits",
	Short: "View credit balance and transactions",
	Long:  "Display your current credit balance and recent transaction history",
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

		var balance struct {
			Balance  float64 `json:"balance"`
			Currency string  `json:"currency"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/credits/balance", &balance); err != nil {
			return fmt.Errorf("fetching balance: %w", err)
		}

		fmt.Printf("Credit Balance: %.2f %s\n\n", balance.Balance, balance.Currency)

		var transactions struct {
			Items []struct {
				ID          string  `json:"id"`
				Amount      float64 `json:"amount"`
				Type        string  `json:"type"`
				Description string  `json:"description"`
				CreatedAt   string  `json:"created_at"`
			} `json:"items"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/credits/transactions?limit=10", &transactions); err != nil {
			// Non-fatal: balance was shown
			return nil
		}

		if len(transactions.Items) > 0 {
			fmt.Println("Recent Transactions:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "DATE\tTYPE\tAMOUNT\tDESCRIPTION")
			for _, t := range transactions.Items {
				fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\n", t.CreatedAt, t.Type, t.Amount, t.Description)
			}
			w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(creditsCmd)
}
