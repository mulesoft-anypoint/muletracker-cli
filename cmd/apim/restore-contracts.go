package apim

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim_contract"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func RestoreAllApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID, contractID string) ([]ApiContracts, error) {
	var allContracts []ApiContracts
	if contractID != "" {
		apiContracts, err := GetSingleApiContracts(ctx, client, orgID, envID, apiID)
		if err != nil {
			return nil, err
		}
		var filteredContracts []apim_contract.ContractDetails
		for _, c := range apiContracts.contracts {
			if strconv.Itoa(int(c.GetId())) == contractID {
				filteredContracts = append(filteredContracts, c)
			}
		}
		apiContracts.contracts = filteredContracts
		allContracts = []ApiContracts{apiContracts}
	} else {
		contracts, err := GetApiContracts(ctx, client, orgID, envID, apiID)
		if err != nil {
			return nil, err
		}
		allContracts = contracts
	}
	for _, contract := range allContracts {
		count := RestoreApiContractsConcurrently(ctx, client, orgID, envID, &contract)
		if count == 0 {
			utils.PrintInfo("No contracts restored for app %s", contract.apiAssetName)
		} else {
			utils.PrintSuccess("Restored %d contracts for app %s", count, contract.apiAssetName)
		}
	}
	return allContracts, nil
}

func RestoreApiContractsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID string, contract *ApiContracts) int {
	const concurrencyLimit = 5
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var lock sync.RWMutex
	count := 0

	// Create a rate limiter ticker
	rateLimiter := time.NewTicker(250 * time.Millisecond)
	defer rateLimiter.Stop()

	for i, c := range contract.contracts {
		wg.Add(1)
		go func(i int, c apim_contract.ContractDetails) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			if strings.EqualFold(c.GetStatus(), "REVOKE") {
				new, err := client.RestoreApiContract(ctx, orgID, envID, strconv.Itoa(int(contract.apiId)), strconv.Itoa(int(c.GetId())))
				if err != nil {
					utils.PrintError("Error restoring contract %d: %v", c.GetId(), err)
					return
				}
				lock.Lock()
				contract.contracts[i] = *new
				count++
				lock.Unlock()
			}
		}(i, c)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return count
}

var restoreContractsCmd = &cobra.Command{
	Use:   "restore-contracts",
	Short: "Restore API contracts",
	Long:  "Restore revoked API contracts",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to restore contracts for all contracts of the API ?")
		}
		if !cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to restore contracts for all contracts of all APIs ?")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		apiID, _ := cmd.Flags().GetString("api")
		contractID, _ := cmd.Flags().GetString("contract")
		token, _ := cmd.Flags().GetString("token")
		output, _ := cmd.Flags().GetString("out")

		if apiID == "" && contractID != "" {
			utils.PrintError("API ID is required when restoring a contract for an API.")
			return
		}

		client, err := anypoint.GetInitializedClient(token)
		if err != nil {
			utils.PrintError("Error initializing client: %v", err)
			return
		}

		orgID, envID, err = utils.ValidateOrgEnv(client, orgID, envID)
		if err != nil {
			utils.PrintError("Validation error: %v", err)
			return
		}

		// Display the client info in a colorful way.
		utils.PrintClientInfo(cmd.Context(), client)

		// Execute restore
		result, err := RestoreAllApiContracts(cmd.Context(), client, orgID, envID, apiID, contractID)
		if err != nil {
			utils.PrintError("Error restoring contracts: %v", err)
			return
		}

		// Export contracts
		if output != "" {
			if err := ExportContracts(output, result); err != nil {
				utils.PrintError("Error exporting contracts: %v", err)
				return
			}
			utils.PrintSuccess("Contracts exported to %s\n", output)
		} else {
			PrintContractsTable(result)
		}
	},
}

func init() {
	restoreContractsCmd.Flags().String("org", "", "Organization ID. If not provided, the id from the saved context will be loaded if present.")
	restoreContractsCmd.Flags().String("env", "", "Environment ID. If not provided, the id from the saved context will be loaded if present.")
	restoreContractsCmd.Flags().String("api", "", "API ID (optional)")
	restoreContractsCmd.Flags().String("contract", "", "Contract ID (optional)")
	restoreContractsCmd.Flags().StringP("token", "t", "", "Anypoint Access Token (optional)")
	// export flags
	restoreContractsCmd.Flags().StringP("out", "o", "", "Export the results to the specified CSV file")
}
