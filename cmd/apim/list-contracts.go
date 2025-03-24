package apim

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim"
	"github.com/mulesoft-anypoint/anypoint-client-go/apim_contract"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

type ApiContracts struct {
	apiId           int32
	apiName         string
	apiTechnology   string
	apiStatus       string
	apiVersion      string
	apiAssetVersion string
	apiAssetId      string
	apiAssetName    string
	contracts       []apim_contract.ContractDetails
}

func GetApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID string) ([]ApiContracts, error) {
	var allContracts []ApiContracts
	if apiID != "" {
		apiContracts, err := GetSingleApiContracts(ctx, client, orgID, envID, apiID)
		if err != nil {
			return nil, err
		}
		allContracts = append(allContracts, apiContracts)
	} else {
		contracts, err := GetApiContractsConcurrently(ctx, client, orgID, envID)
		if err != nil {
			return nil, err
		}
		allContracts = append(allContracts, contracts...)
	}
	return allContracts, nil
}

func GetSingleApiContracts(ctx context.Context, client *anypoint.Client, orgID, envID, apiID string) (ApiContracts, error) {
	api, err := client.GetApi(ctx, orgID, envID, apiID)
	if err != nil {
		utils.PrintError("Error getting API: %v", err)
		return ApiContracts{}, err
	}
	asset, err := client.GetApiAsset(ctx, orgID, envID, apiID)
	if err != nil {
		utils.PrintError("Error getting asset: %v", err)
		return ApiContracts{}, err
	}
	contracts, err := GetContractsDetailsConcurrently(ctx, client, orgID, envID, apiID)
	if err != nil {
		utils.PrintError("Error getting contracts for API %s: %v", apiID, err)
		return ApiContracts{}, err
	}
	return ApiContracts{
		apiId:           api.GetId(),
		apiName:         asset.GetName(),
		apiVersion:      api.GetProductVersion(),
		apiTechnology:   api.GetTechnology(),
		apiStatus:       api.GetStatus(),
		apiAssetVersion: api.GetAssetVersion(),
		apiAssetId:      api.GetAssetId(),
		apiAssetName:    asset.GetName(),
		contracts:       contracts,
	}, nil
}

func GetApiContractsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID string) ([]ApiContracts, error) {
	const concurrencyLimit = 10
	var allContracts []ApiContracts
	assets, err := client.GetAllApis(ctx, orgID, envID)
	if err != nil {
		utils.PrintError("Error getting APIs: %v", err)
		return nil, err
	}
	utils.PrintInfo("Total APIs: %d", CountApis(assets))
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	resultsCh := make(chan *ApiContracts, CountApis(assets))

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for _, asset := range assets {
		for _, api := range asset.GetApis() {
			wg.Add(1)
			go func(asset apim.ApimInstanceCollectionAssetsInner, api apim.ApimInstanceCollectionAssetsInnerApisInner) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore.
				defer func() { <-sem }() // Release semaphore.
				<-rateLimiter.C          // Wait for rate limiter tick.
				contracts, err := GetContractsDetailsConcurrently(ctx, client, orgID, envID, strconv.Itoa(int(api.GetId())))
				if err != nil {
					utils.PrintError("Error getting contracts for API %s: %v", strconv.Itoa(int(api.GetId())), err)
					resultsCh <- nil
					return
				}
				resultsCh <- &ApiContracts{
					apiId:           api.GetId(),
					apiName:         strings.TrimSpace(asset.GetExchangeAssetName()),
					apiTechnology:   api.GetTechnology(),
					apiStatus:       api.GetStatus(),
					apiVersion:      api.GetProductVersion(),
					apiAssetVersion: api.GetAssetVersion(),
					apiAssetId:      api.GetAssetId(),
					apiAssetName:    strings.TrimSpace(asset.GetExchangeAssetName()),
					contracts:       contracts,
				}
			}(asset, api)
		}
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsCh)

	// Collect results
	for details := range resultsCh {
		if details != nil {
			allContracts = append(allContracts, *details)
		}
	}

	return allContracts, nil
}

func GetContractsDetailsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID, apiID string) ([]apim_contract.ContractDetails, error) {
	const concurrencyLimit = 10
	contracts, err := client.GetApiContracts(ctx, orgID, envID, apiID)
	if err != nil {
		utils.PrintError("Error getting contracts for API %s: %v", apiID, err)
		return nil, err
	}
	var allContracts []apim_contract.ContractDetails
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	resultsCh := make(chan *apim_contract.ContractDetails, len(contracts))

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	//for each contact get details
	for _, contract := range contracts {
		wg.Add(1)
		go func(contract apim_contract.Contract) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			details, err := client.GetContractDetails(ctx, orgID, envID, apiID, strconv.Itoa(int(contract.GetId())))
			if err != nil {
				utils.PrintError("Error getting contract details for API %s: %v", apiID, err)
				resultsCh <- nil
				return
			}
			resultsCh <- details
		}(contract)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsCh)

	// Collect results
	for details := range resultsCh {
		if details != nil {
			allContracts = append(allContracts, *details)
		}
	}

	return allContracts, nil
}

func GetContractLastUpdatedDate(contract *apim_contract.ContractDetails) time.Time {
	layout := "2006-01-02T15:04:05Z0700"
	audit := contract.GetAudit()
	createDate := audit.GetCreated().Date
	var lastUpdate time.Time
	status := contract.GetStatus()
	if strings.EqualFold(status, "REVOKED") {
		if revokedDate, ok := contract.GetRevokedDateOk(); ok && revokedDate != nil {
			lastUpdate, _ = time.Parse(layout, *revokedDate)
		}
	} else if strings.EqualFold(status, "APPROVED") {
		if approvedDate, ok := contract.GetApprovedDateOk(); ok && approvedDate != nil {
			lastUpdate, _ = time.Parse(layout, *approvedDate)
		}
	} else if strings.EqualFold(status, "PENDING") {
		if createDate != nil {
			lastUpdate, _ = time.Parse(layout, *createDate)
		}
	} else if strings.EqualFold(status, "REJECTED") {
		if rejectedDate, ok := contract.GetRejectedDateOk(); ok && rejectedDate != nil {
			lastUpdate, _ = time.Parse(layout, *rejectedDate)
		}
	} else {
		if createDate != nil {
			lastUpdate, _ = time.Parse(layout, *createDate)
		}
	}
	return lastUpdate
}

func CountContracts(apiContracts []ApiContracts) int {
	count := 0
	for _, apiContract := range apiContracts {
		count += len(apiContract.contracts)
	}
	return count
}

func Contract2Map(apiContracts []ApiContracts) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	maxOwner := 0
	for _, apiContract := range apiContracts {
		for _, contract := range apiContract.contracts {
			application := contract.GetApplication()
			owners := application.GetOwners()
			lastUpdate := GetContractLastUpdatedDate(&contract)
			item := map[string]any{
				"API ID":                         apiContract.apiId,
				"Api Name":                       apiContract.apiAssetName,
				"Api Status":                     apiContract.apiStatus,
				"Api Version":                    apiContract.apiVersion,
				"Api Asset Version":              apiContract.apiAssetVersion,
				"Api Asset ID":                   apiContract.apiAssetId,
				"Contract ID":                    contract.GetId(),
				"Contract Application Client ID": application.GetClientId(),
				"Contract Application Name":      application.GetName(),
				"Contract Status":                contract.GetStatus(),
				"Contract LastUpdate":            lastUpdate.Local().Format("2006-01-02 15:04:05"),
			}
			for i, owner := range owners {
				item["Contract Owner #"+strconv.Itoa(i+1)+" Name"] = owner.GetFirstName() + " " + owner.GetLastName()
				item["Contract Owner #"+strconv.Itoa(i+1)+" Email"] = owner.GetEmail()
				if i+1 > maxOwner {
					maxOwner = i + 1
				}
			}
			data = append(data, item)
		}
	}

	order := []string{"API ID", "Api Name", "Api Status", "Api Version", "Api Asset Version", "Api Asset ID", "Contract ID", "Contract Application Client ID", "Contract Application Name", "Contract Status", "Contract LastUpdate"}
	for i := 1; i <= maxOwner; i++ {
		order = append(order, "Contract Owner #"+strconv.Itoa(i)+" Name", "Contract Owner #"+strconv.Itoa(i)+" Email")
	}

	return data, order
}

func PrintContractsTable(apiContracts []ApiContracts) {
	data, order := Contract2Map(apiContracts)
	utils.PrintGenericTable(data, order)
}

func ExportContracts(path string, apiContracts []ApiContracts) error {
	data, order := Contract2Map(apiContracts)
	return utils.ExportGenericCSV(path, data, order)
}

var listContractsCmd = &cobra.Command{
	Use:   "list-contracts",
	Short: "List API contracts",
	Long:  "List all contracts for a specific application or all applications in an organization/environment",
	Run: func(cmd *cobra.Command, args []string) {
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		apiID, _ := cmd.Flags().GetString("api")
		token, _ := cmd.Flags().GetString("token")
		output, _ := cmd.Flags().GetString("output")

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

		allContracts, err := GetApiContracts(cmd.Context(), client, orgID, envID, apiID)
		if err != nil {
			utils.PrintError("Error getting contracts: %v", err)
			return
		}

		if len(allContracts) == 0 {
			fmt.Println("No contracts found")
			return
		}

		if output != "" {
			if err := ExportContracts(output, allContracts); err != nil {
				utils.PrintError("Error exporting contracts: %v", err)
			}
			fmt.Println("Contracts exported to", output)
		} else {
			PrintContractsTable(allContracts)
		}
	},
}

func init() {
	listContractsCmd.Flags().String("org", "", "Business Group ID")
	listContractsCmd.Flags().String("env", "", "Environment ID")
	listContractsCmd.Flags().String("api", "", "API ID (optional)")
	listContractsCmd.Flags().StringP("token", "t", "", "Anypoint Access Token (optional)")
	listContractsCmd.Flags().StringP("output", "o", "", "Output file (optional)")
}
