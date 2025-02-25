package exchange

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func deleteClientAppsConcurrently(ctx context.Context, client *anypoint.Client, orgID string, results []ClientAppResult) {
	const concurrencyLimit = 5
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()
	for _, res := range results {
		wg.Add(1)
		go func(res ClientAppResult) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			if err := client.DeleteExchangeClientApp(ctx, orgID, res.ClientApp.GetId()); err != nil {
				utils.PrintError("Error while deleting client app %s: %v\v", res.ClientApp.GetName(), err)
			} else {
				fmt.Printf("Deleted client app %s.\n", res.ClientApp.GetName())
			}
		}(res)
	}
	wg.Wait()
}

func deleteClientAppsWithEmptyContracts(ctx context.Context, orgID string, client *anypoint.Client) {
	//Get All exchange client apps
	list, err := client.GetExchangeClientApps(ctx, orgID, true)
	if err != nil {
		utils.PrintError("Error retrieving Exchange Client Apps %v/n", err)
		return
	}
	//Get All exchange client apps contracts and filter them
	allResults := ListExchClientAppsConcurrently(ctx, client, orgID, list)
	fmt.Printf("* Collected contract data for %d apps.\n", len(allResults))
	finalResults := FilterClientAppResults(allResults, "empty")
	fmt.Printf("* After applying filter '%s', %d client apps remain.\n", "empty", len(finalResults))
	fmt.Println()
	if len(finalResults) == 0 {
		fmt.Println("No apps match the filter criteria.")
		return
	}
	deleteClientAppsConcurrently(ctx, client, orgID, finalResults)
}

var deleteClientAppCmd = &cobra.Command{
	Use:   "delete-client-app",
	Short: "Delete an Exchange client application",
	Long:  `Delete an Exchange client application. You must provide the application ID or name.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		ctx := cmd.Context()
		// Retrieve flags.
		orgID, _ := cmd.Flags().GetString("org")
		appIDStr, _ := cmd.Flags().GetString("id")
		adminToken, _ := cmd.Flags().GetString("admin-token")
		filterByEmptyContract, _ := cmd.Flags().GetBool("with-empty-contract")
		// Validate params
		if appIDStr == "" && !filterByEmptyContract {
			utils.PrintError("Error: Application ID or filter is required to delete Exchange Client Applications.")
			return
		}
		// Retrieve the authenticated client.
		var client *anypoint.Client
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
		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)
		if filterByEmptyContract {
			deleteClientAppsWithEmptyContracts(ctx, orgID, client)
		} else {
			appID, err := strconv.Atoi(appIDStr)
			if err != nil {
				utils.PrintError("Error: Application ID is unvalid.")
				return
			}
			if err := client.DeleteExchangeClientApp(ctx, orgID, int32(appID)); err != nil {
				utils.PrintError("Error deleting the application client %v\n", err)
				return
			}
			fmt.Printf("Deleted Exchange application with ID: %s\n", appIDStr)
		}
	},
}

func init() {
	deleteClientAppCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	deleteClientAppCmd.Flags().String("id", "", "ID of the Exchange application to delete (optional). Either this flag should be present or the empty contract should be present.")
	deleteClientAppCmd.Flags().StringP("admin-token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all the org's client applications.")
	deleteClientAppCmd.Flags().Bool("with-empty-contract", false, "ID of the Exchange application to delete (optional). Either this flag should be present or the id of the app you want to delete.")
}
