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

func RevokeAllApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID, contractID string) ([]ApiContracts, error) {
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
		count := RevokeApiContractsConcurrently(ctx, client, orgID, envID, &contract)
		if count == 0 {
			utils.PrintInfo("No contracts revoked for app %s", contract.apiAssetName)
		} else {
			utils.PrintSuccess("Revoked %d contracts for app %s", count, contract.apiAssetName)
		}
	}
	return allContracts, nil
}

func RevokeApiContractsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID string, contract *ApiContracts) int {
	const concurrencyLimit = 10
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var lock sync.RWMutex

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	count := 0

	//for each contact revoke
	for i, c := range contract.contracts {
		wg.Add(1)
		go func(i int, c apim_contract.ContractDetails) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			if strings.EqualFold(c.GetStatus(), "APPROVED") {
				new, err := client.RevokeApiContract(ctx, orgID, envID, strconv.Itoa(int(contract.apiId)), strconv.Itoa(int(c.GetId())))
				if err != nil {
					utils.PrintError("Error revoking app %s contract %d: %v", contract.apiAssetName, c.GetId(), err)
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

var revokeContractsCmd = &cobra.Command{
	Use:   "revoke-contracts",
	Short: "Revoke API contracts",
	Long:  "Revoke contracts for a specific application in an organization/environment",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to revoke all contracts for this API?")
		}
		if !cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to revoke all contracts for all APIs?")
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
			utils.PrintError("API ID is required when revoking a contract for an API.")
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

		allContracts, err := RevokeAllApiContracts(cmd.Context(), client, orgID, envID, apiID, contractID)
		if err != nil {
			utils.PrintError("Error revoking contracts: %v", err)
			return
		}

		if output != "" {
			if err := ExportContracts(output, allContracts); err != nil {
				utils.PrintError("Error exporting contracts: %v", err)
			}
			utils.PrintSuccess("Contracts exported to %s", output)
		} else {
			PrintContractsTable(allContracts)
		}

	},
}

func init() {
	revokeContractsCmd.Flags().String("org", "", "Organization ID. If not provided, the id from the saved context will be loaded if present.")
	revokeContractsCmd.Flags().String("env", "", "Environment ID. If not provided, the id from the saved context will be loaded if present.")
	revokeContractsCmd.Flags().String("api", "", "API ID (optional)")
	revokeContractsCmd.Flags().String("contract", "", "Contract ID (optional)")
	revokeContractsCmd.Flags().StringP("token", "t", "", "Anypoint Access Token (optional)")
	// export flags
	revokeContractsCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
}
