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

var invoicesCmd = &cobra.Command{
	Use:   "invoices",
	Short: "List billing invoices",
	Long:  "Display your billing invoices and payment history",
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

		var invoices struct {
			Items []struct {
				ID        string  `json:"id"`
				Amount    float64 `json:"amount"`
				Currency  string  `json:"currency"`
				Status    string  `json:"status"`
				Period    string  `json:"period"`
				CreatedAt string  `json:"created_at"`
				PaidAt    string  `json:"paid_at,omitempty"`
			} `json:"items"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/invoices", &invoices); err != nil {
			return fmt.Errorf("fetching invoices: %w", err)
		}

		if len(invoices.Items) == 0 {
			fmt.Println("No invoices found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "DATE\tAMOUNT\tSTATUS\tPERIOD")
		for _, inv := range invoices.Items {
			fmt.Fprintf(w, "%s\t%.2f %s\t%s\t%s\n",
				inv.CreatedAt, inv.Amount, inv.Currency, inv.Status, inv.Period)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(invoicesCmd)
}
