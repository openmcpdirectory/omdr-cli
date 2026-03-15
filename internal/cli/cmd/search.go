package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/defaults"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	searchNamespace string
	searchMinScore  int
	searchVerified  bool
	searchLimit     int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for MCP servers",
	Long:  "Search for MCP servers using natural language queries with semantic search",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		// Get API client
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = defaults.RegistryURL
		}

		apiClient := client.NewClient(apiURL)

		// Build search request
		searchReq := entity.SearchQuery{
			Query: query,
			Limit: searchLimit,
		}

		if searchNamespace != "" {
			searchReq.Namespace = &searchNamespace
		}

		if cmd.Flags().Changed("min-score") {
			searchReq.MinScore = &searchMinScore
		}

		searchReq.VerifiedOnly = searchVerified

		// Execute search
		if verbose {
			fmt.Printf("Searching for: %s\n", query)
		}

		var response struct {
			Results []entity.SearchResult `json:"results"`
			Query   string                `json:"query"`
			Limit   int                   `json:"limit"`
			Offset  int                   `json:"offset"`
		}
		if err := apiClient.Post(cmd.Context(), "/api/v1/search", searchReq, &response); err != nil {
			// Check for rate limiting
			if isRateLimited, retryAfter := apiClient.IsRateLimited(err); isRateLimited {
				return fmt.Errorf("rate limited. Please retry after %v", retryAfter)
			}
			return fmt.Errorf("search failed: %w", err)
		}

		// Handle empty results
		if len(response.Results) == 0 {
			fmt.Println("No servers found matching your query.")
			fmt.Println("\nTry:")
			fmt.Println("  - Using different search terms")
			fmt.Println("  - Removing filters (--namespace, --min-score, --verified)")
			fmt.Println("  - Searching for broader topics")
			return nil
		}

		// Display results in formatted table
		displaySearchResults(response.Results)

		return nil
	},
}

func displaySearchResults(results []entity.SearchResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tNAMESPACE\tTRUST SCORE\tINSTALLS\tDESCRIPTION")
	fmt.Fprintln(w, "----\t---------\t-----------\t--------\t-----------")

	// Results
	for _, result := range results {
		server := result.Server

		// Truncate description if too long
		description := server.Description
		if len(description) > 60 {
			description = description[:57] + "..."
		}

		// Format verified indicator
		verifiedMark := ""
		if server.Verified {
			verifiedMark = "✓ "
		}

		fmt.Fprintf(w, "%s%s\t%s\t%d\t%d\t%s\n",
			verifiedMark,
			server.Name,
			server.Namespace,
			server.TrustScore,
			server.InstallCount,
			description,
		)
	}

	fmt.Fprintln(w)
	fmt.Printf("Found %d server(s)\n", len(results))
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchNamespace, "namespace", "", "Filter by namespace")
	searchCmd.Flags().IntVar(&searchMinScore, "min-score", 0, "Minimum trust score (0-100)")
	searchCmd.Flags().BoolVar(&searchVerified, "verified", false, "Show only verified servers")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum number of results")
}
