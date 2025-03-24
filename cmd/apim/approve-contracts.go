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

func ApproveAllApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID, contractID string) ([]ApiContracts, error) {
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
		count := ApproveApiContractsConcurrently(ctx, client, orgID, envID, &contract)
		if count == 0 {
			utils.PrintInfo("No contracts approved for app %s", contract.apiAssetName)
		} else {
			utils.PrintSuccess("Approved %d contracts for app %s", count, contract.apiAssetName)
		}
	}
	return allContracts, nil
}

func ApproveApiContractsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID string, contract *ApiContracts) int {
	const concurrencyLimit = 10
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var lock sync.RWMutex
	count := 0

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for i, c := range contract.contracts {
		wg.Add(1)
		go func(i int, c apim_contract.ContractDetails) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			if strings.EqualFold(c.GetStatus(), "REJECTED") || strings.EqualFold(c.GetStatus(), "PENDING") {
				new, err := client.ApproveApiContract(ctx, orgID, envID, strconv.Itoa(int(contract.apiId)), strconv.Itoa(int(c.GetId())))
				if err != nil {
					utils.PrintError("Error approving contract %d: %v", c.GetId(), err)
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

var approveContractsCmd = &cobra.Command{
	Use:   "approve-contracts",
	Short: "Approve API contracts",
	Long:  "Approve revoked or pending API contracts",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to approve rejected and pending contracts for all contracts of the API?")
		}
		if !cmd.Flags().Changed("api") && !cmd.Flags().Changed("contract") {
			return utils.GetConfirmed("Are you sure you want to approve rejected and pending contracts for all contracts of all APIs?")
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
			utils.PrintError("API ID is required when approving a contract for an API.")
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
		result, err := ApproveAllApiContracts(cmd.Context(), client, orgID, envID, apiID, contractID)
		if err != nil {
			utils.PrintError("Error approving contracts: %v", err)
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
	approveContractsCmd.Flags().String("org", "", "Organization ID. If not provided, the id from the saved context will be loaded if present.")
	approveContractsCmd.Flags().String("env", "", "Environment ID. If not provided, the id from the saved context will be loaded if present.")
	approveContractsCmd.Flags().String("api", "", "API ID (optional)")
	approveContractsCmd.Flags().String("contract", "", "Contract ID (optional)")
	approveContractsCmd.Flags().StringP("token", "t", "", "Anypoint Access Token (optional)")
	// export flags
	approveContractsCmd.Flags().StringP("out", "o", "", "Export the results to the specified CSV file")
}
