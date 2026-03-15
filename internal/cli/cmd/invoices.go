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
			apiURL = defaults.CLIURL
		}
		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		var invoices []struct {
			ID         string `json:"id"`
			Amount     int    `json:"amount"`
			Status     string `json:"status"`
			CreatedAt  string `json:"createdAt"`
			PaidAt     string `json:"paidAt,omitempty"`
			InvoiceURL string `json:"invoiceUrl,omitempty"`
		}
		if err := apiClient.Get(cmd.Context(), "/api/v1/users/me/invoices", &invoices); err != nil {
			return fmt.Errorf("fetching invoices: %w", err)
		}

		if len(invoices) == 0 {
			fmt.Println("No invoices found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "DATE\tAMOUNT\tSTATUS")
		for _, inv := range invoices {
			fmt.Fprintf(w, "%s\t%d\t%s\n", inv.CreatedAt, inv.Amount, inv.Status)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(invoicesCmd)
}
