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
			UserID  string `json:"user_id"`
			Balance int64  `json:"balance"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/credits/balance", &balance); err != nil {
			return fmt.Errorf("fetching balance: %w", err)
		}

		fmt.Printf("Credit Balance: %d credits\n\n", balance.Balance)

		var txnResp struct {
			Transactions []struct {
				ID          string `json:"id"`
				Amount      int64  `json:"amount"`
				Type        string `json:"type"`
				Description string `json:"description"`
				CreatedAt   string `json:"created_at"`
			} `json:"transactions"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/credits/transactions?limit=10", &txnResp); err != nil {
			// Non-fatal: balance was shown
			return nil
		}

		if len(txnResp.Transactions) > 0 {
			fmt.Println("Recent Transactions:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "DATE\tTYPE\tAMOUNT\tDESCRIPTION")
			for _, t := range txnResp.Transactions {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", t.CreatedAt, t.Type, t.Amount, t.Description)
			}
			w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(creditsCmd)
}
