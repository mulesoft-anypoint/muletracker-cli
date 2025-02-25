package exchange

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/exchange_client_apps"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

type ClientAppResult struct {
	ClientApp *exchange_client_apps.ClientApp
	Contracts []exchange_client_apps.ClientAppContract
	Err       error
}

func ListExchClientAppContracts(ctx context.Context, client *anypoint.Client, orgID string, clientApp *exchange_client_apps.ClientApp) ClientAppResult {
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
func ListExchClientAppsConcurrently(ctx context.Context, client *anypoint.Client, orgID string, clientApps []exchange_client_apps.ClientApp) []ClientAppResult {
	const concurrencyLimit = 5
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	resultsCh := make(chan ClientAppResult, len(clientApps))

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for _, clientApp := range clientApps {
		wg.Add(1)
		go func(app exchange_client_apps.ClientApp) {
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
func CountContractsByStatus(contracts []exchange_client_apps.ClientAppContract) map[string]int {
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
func FilterClientAppResults(results []ClientAppResult, filterFlag string) []ClientAppResult {
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

func ClientAppResult2Map(results []ClientAppResult) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	for _, r := range results {
		total := len(r.Contracts)
		countMap := CountContractsByStatus(r.Contracts)
		approved := 0
		revoked := 0
		pending := 0
		if val, ok := countMap["APPROVED"]; ok {
			approved = val
		}
		if val, ok := countMap["REVOKED"]; ok {
			revoked = val
		}
		if val, ok := countMap["PENDING"]; ok {
			pending = val
		}
		data = append(data, map[string]any{
			"App ID":             r.ClientApp.GetId(),
			"App Name":           r.ClientApp.GetName(),
			"Client ID":          r.ClientApp.GetClientId(),
			"Total Contracts":    total,
			"Approved Contracts": approved,
			"Revoked Contracts":  revoked,
			"Pending Contracts":  pending,
		})
	}
	order := []string{"App ID", "App Name", "Client ID", "Total Contracts", "Approved Contracts", "Revoked Contracts", "Pending Contracts"}

	return data, order
}

func PrintClientAppsSummaryTable(results []ClientAppResult) {
	data, order := ClientAppResult2Map(results)
	utils.PrintGenericTable(data, order)
}

func ExportClientAppsSummaryTable(fileName string, results []ClientAppResult) error {
	data, order := ClientAppResult2Map(results)
	return utils.ExportGenericCSV(fileName, data, order)
}

var listClientAppsCmd = &cobra.Command{
	Use:   "list-client-apps",
	Short: "List Exchange Client applications",
	Long: `List all or parts of Exchange client applications.
		If you need to get all the available exchange apps on your organization (not just the client apps created by the user making the Query).
    You need to use this call with your Master Org id, a bearer token for an Admin user, and the query parameter 'targetAdminSite' set to 'true'. This call will return every application (with pagination if more than the set limit) for this particular Anypoint Account.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flags.
		filterContract, _ := cmd.Flags().GetString("filter-contract")
		orgID, _ := cmd.Flags().GetString("org")
		adminToken, _ := cmd.Flags().GetString("admin-token")
		exportFile, _ := cmd.Flags().GetString("out")
		// Retrieve the authenticated client.
		var client *anypoint.Client
		var err error
		if adminToken != "" {
			client, err = anypoint.GetClientFromContext(anypoint.WithSkipTokenExpiration())
			if err != nil {
				utils.PrintError("Error retrieving client: %v\n", err)
				return
			}
			client.SetAdminAccessToken(adminToken)
		} else {
			client, err = anypoint.GetClientFromContext()
			if err != nil {
				utils.PrintError("Error retrieving client: %v\n", err)
				return
			}
		}
		//Read Org ID
		if client.IsOrgEmpty() && orgID == "" {
			utils.PrintError("Please provide --org flag")
			return
		}
		if orgID == "" {
			orgID = client.Org
		} else {
			client.SetOrg(orgID)
		}
		//Get All exchange client apps
		list, err := client.GetExchangeClientApps(ctx, orgID, true)
		if err != nil {
			utils.PrintError("Error retrieving Exchange Client Apps %v/n", err)
			return
		}
		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)
		//Get All exchange client apps contracts
		allResults := ListExchClientAppsConcurrently(ctx, client, orgID, list)
		fmt.Printf("* Collected contract data for %d apps.\n", len(allResults))
		// Apply filter.
		finalResults := FilterClientAppResults(allResults, filterContract)
		fmt.Printf("* After applying filter '%s', %d client apps remain.\n", filterContract, len(finalResults))
		if len(finalResults) == 0 {
			fmt.Println("No apps match the filter criteria.")
			return
		}
		// If export flag is provided, export results to CSV.
		if exportFile != "" {
			err := ExportClientAppsSummaryTable(exportFile, finalResults)
			if err != nil {
				utils.PrintError("Error exporting results to CSV: %v\n", err)
				return
			}
			fmt.Printf("\nResults successfully exported to %s\n", exportFile)
		} else {
			// Otherwise, print a summary table.
			PrintClientAppsSummaryTable(finalResults)
		}
	},
}

func init() {
	listClientAppsCmd.Flags().String("org", "", "The Business Group ID. This should be the root org id")
	listClientAppsCmd.Flags().StringP("admin-token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all the org's client applications")
	//Filters
	listClientAppsCmd.Flags().String("filter-contract", "all", "Filter results: all (default), nonempty (only client apps with contracts), or empty (only client apps with no contracts)")
	// export flags
	listClientAppsCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
}
