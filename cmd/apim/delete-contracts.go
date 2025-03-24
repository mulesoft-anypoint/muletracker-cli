package apim

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim_contract"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func DeleteApiContractsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID, apiID string, contracts []ApiContracts) error {
	const concurrencyLimit = 10
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for _, contract := range contracts {
		for _, c := range contract.contracts {
			wg.Add(1)
			go func(c apim_contract.ContractDetails) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				<-rateLimiter.C

				err := client.DeleteApiContract(ctx, orgID, envID, apiID, strconv.Itoa(int(c.GetId())))
				if err != nil {
					select {
					case errCh <- fmt.Errorf("error deleting contract %d: %w", c.GetId(), err):
					default:
					}
					return
				}
				utils.PrintSuccess("Deleted contract ID %d", c.GetId())
			}(c)
		}
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

func DeleteApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID string) ([]ApiContracts, error) {
	contracts, err := GetApiContracts(ctx, client, orgID, envID, apiID)
	if err != nil {
		utils.PrintError("Error fetching contracts: %v", err)
		return nil, err
	}

	err = DeleteApiContractsConcurrently(ctx, client, orgID, envID, apiID, contracts)
	if err != nil {
		utils.PrintError("Deletion completed with errors: %v", err)
		return nil, err
	}
	return contracts, nil
}

var deleteContractsCmd = &cobra.Command{
	Use:   "delete-contracts",
	Short: "Permanently delete API contracts",
	Long:  "Delete revoked contracts for a specific API in an organization/environment",
	PreRun: func(cmd *cobra.Command, args []string) {
		if utils.GetConfirmed("WARNING: This will permanently delete contracts. Continue?") != nil {
			utils.PrintError("Aborted by user")
			return
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		apiID, _ := cmd.Flags().GetString("api")
		contractID, _ := cmd.Flags().GetString("contract")
		token, _ := cmd.Flags().GetString("token")
		output, _ := cmd.Flags().GetString("out")

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

		// Print client info
		utils.PrintClientInfo(cmd.Context(), client)

		if contractID != "" {
			err := client.DeleteApiContract(cmd.Context(), orgID, envID, apiID, contractID)
			if err != nil {
				utils.PrintError("Error deleting contract: %v", err)
				return
			}
			utils.PrintSuccess("Deleted contract ID %s", contractID)
			return
		}

		contracts, err := DeleteApiContracts(cmd.Context(), client, orgID, envID, apiID)
		if err != nil {
			utils.PrintError("Deletion completed with errors: %v", err)
			return
		}
		utils.PrintSuccess("Successfully deleted %d contracts", CountContracts(contracts))

		if output != "" {
			if err := ExportContracts(output, contracts); err != nil {
				utils.PrintError("Error exporting contracts: %v", err)
			}
			utils.PrintSuccess("Contracts exported to %s", output)
		} else {
			PrintContractsTable(contracts)
		}
	},
}

func init() {
	deleteContractsCmd.Flags().String("org", "", "Organization ID")
	deleteContractsCmd.Flags().String("env", "", "Environment ID")
	deleteContractsCmd.Flags().String("api", "", "API ID (required)")
	deleteContractsCmd.Flags().String("contract", "", "Contract ID (optional)")
	deleteContractsCmd.Flags().StringP("token", "t", "", "Anypoint Access Token (optional)")
	// export flags
	deleteContractsCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
	//required flags
	deleteContractsCmd.MarkFlagRequired("api")
}
