package exchange

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/exchange_apps"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/spf13/cobra"
)

type ClientAppResult struct {
	ClientApp *exchange_apps.GetExchangeAppsResponseInner
	Contracts []exchange_apps.GetExchangeAppContractsResponseInner
	Err       error
}

func ListExchClientAppContracts(ctx context.Context, client *anypoint.Client, orgID string, clientApp *exchange_apps.GetExchangeAppsResponseInner) ClientAppResult {
	var result ClientAppResult
	result.ClientApp = clientApp
	contracts, err := client.GetExchangeClientAppContracts(ctx, orgID, clientApp.GetId())
	if err != nil {
		result.Err = err
	} else {
		result.Contracts = contracts
	}

	return result
}

// monitorAppsConcurrently monitors a list of apps with concurrency and rate limiting.
func ListExchClientAppsConcurrently(ctx context.Context, client *anypoint.Client, orgID string, clientApps []exchange_apps.GetExchangeAppsResponseInner) []ClientAppResult {
	const concurrencyLimit = 5
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	resultsCh := make(chan ClientAppResult, len(clientApps))

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for _, clientApp := range clientApps {
		wg.Add(1)
		go func(app exchange_apps.GetExchangeAppsResponseInner) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			result := ListExchClientAppContracts(ctx, client, orgID, &app)
			resultsCh <- result
		}(clientApp)
	}

	wg.Wait()
	close(resultsCh)

	var results []ClientAppResult
	for r := range resultsCh {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "Error reading client app %d: %v\n", r.ClientApp.GetId(), r.Err)
		}
		results = append(results, r)
	}
	return results
}

// Returns the count of contracts by status
func countContractsByStatus(contracts []exchange_apps.GetExchangeAppContractsResponseInner) map[string]int {
	data := make(map[string]int)
	for _, contract := range contracts {
		if val, ok := data[contract.GetStatus()]; ok {
			data[contract.GetStatus()] = val + 1
		} else {
			data[contract.GetStatus()] = 1
		}
	}
	return data
}

// filterClientAppResults applies the filter flag to the full list of results.
// filterFlag can be: "all", "nonempty", or "empty".
func filterClientAppResults(results []ClientAppResult, filterFlag string) []ClientAppResult {
	var filtered []ClientAppResult
	switch strings.ToLower(filterFlag) {
	case "nonempty":
		for _, r := range results {
			if len(r.Contracts) > 0 {
				filtered = append(filtered, r)
			}
		}
	case "empty":
		for _, r := range results {
			if len(r.Contracts) == 0 {
				filtered = append(filtered, r)
			}
		}
	default:
		// "all" or any other value returns all results.
		filtered = results
	}
	return filtered
}

// printAppsSummaryTable prints a condensed table of app monitoring results
// using tabwriter for alignment.
func printClientAppsSummaryTable(results []ClientAppResult) {
	// Create a new tabwriter with a minimum width of 0, tab width of 8,
	// padding of 2, and using a tab ('\t') as the padding character.
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header row.
	fmt.Println()
	fmt.Fprintln(w, "App ID\tApp Name\tClient Id\tContracts")
	fmt.Fprintln(w, "------\t--------\t---------\t---------")

	// Iterate over the results and print each row.
	for _, r := range results {
		l := len(r.Contracts)
		contractData := "empty"
		if l > 0 {
			countMap := countContractsByStatus(r.Contracts)
			arr := []string{fmt.Sprintf("Total %d", l)}
			for k, v := range countMap {
				arr = append(arr, fmt.Sprintf("%s %d", k, v))
			}
			contractData = strings.Join(arr[:], " / ")
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", r.ClientApp.GetId(), r.ClientApp.GetName(), r.ClientApp.GetClientId(), contractData)
	}

	// Flush the writer to ensure output is written.
	w.Flush()
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Exchange applications",
	Long: `List all or parts of Exchange client applications.
		If you need to get all the available exchange apps on your organization (not just the client apps created by the user making the Query).
    You need to use this call with your Master Org id, a bearer token for an Admin user, and the query parameter 'targetAdminSite' set to 'true'. This call will return every application (with pagination if more than the set limit) for this particular Anypoint Account.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flags.
		filterContract, _ := cmd.Flags().GetString("filter-contract")
		orgID, _ := cmd.Flags().GetString("org")
		adminToken, _ := cmd.Flags().GetString("adminToken")

		// Retrieve the authenticated client.
		var client *anypoint.Client
		var err error
		if adminToken != "" {
			client, err = anypoint.GetClientFromContext(anypoint.WithSkipTokenExpiration())
			if err != nil {
				fmt.Printf("Error retrieving client: %v\n", err)
				return
			}
			client.SetAdminAccessToken(adminToken)
		} else {
			client, err = anypoint.GetClientFromContext()
			if err != nil {
				fmt.Printf("Error retrieving client: %v\n", err)
				return
			}
		}
		// Save/Load org and env
		if client.IsOrgEmpty() && orgID == "" {
			fmt.Println("Please provide --org flag")
			return
		}
		if orgID == "" {
			orgID = client.Org
		}
		//Get All exchange client apps
		list, err := client.GetExchangeClientApps(ctx, orgID, true)
		if err != nil {
			fmt.Printf("Error retrieving Exchange Client Apps %v/n", err)
			return
		}
		// Display the client info in a colorful way.
		PrintClientInfo(ctx, client)
		//Get All exchange client apps contracts
		allResults := ListExchClientAppsConcurrently(ctx, client, orgID, list)
		fmt.Printf("* Collected contract data for %d apps.\n", len(allResults))
		// Apply filter.
		finalResults := filterClientAppResults(allResults, filterContract)
		fmt.Printf("* After applying filter '%s', %d client apps remain.\n", filterContract, len(finalResults))
		if len(finalResults) == 0 {
			fmt.Println("No apps match the filter criteria.")
			return
		}

		printClientAppsSummaryTable(finalResults)
	},
}

func init() {
	listCmd.Flags().StringP("org", "o", "", "The Business Group ID. This should be the root org id")
	listCmd.Flags().StringP("adminToken", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all the org's client applications")
	//Filters
	listCmd.Flags().String("filter-contract", "all", "Filter results: all (default), nonempty (only client apps with contracts), or empty (only client apps with no contracts)")
}
