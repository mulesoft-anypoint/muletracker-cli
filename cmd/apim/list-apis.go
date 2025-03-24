package apim

import (
	"strings"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func FilterApis(instances []apim.ApimInstanceCollectionAssetsInner, fn func(apim.ApimInstanceCollectionAssetsInnerApisInner) bool) []apim.ApimInstanceCollectionAssetsInner {
	var filtered []apim.ApimInstanceCollectionAssetsInner
	for _, inst := range instances {
		apiRes := make([]apim.ApimInstanceCollectionAssetsInnerApisInner, 0)
		for _, api := range inst.GetApis() {
			// We compare ignoring case.
			if fn(api) {
				apiRes = append(apiRes, api)
			}
		}
		if len(apiRes) > 0 {
			inst.SetApis(apiRes)
			inst.SetTotalApis(int32(len(apiRes)))
			filtered = append(filtered, inst)
		}
	}
	return filtered
}

func FilterApiByType(instances []apim.ApimInstanceCollectionAssetsInner, typ string) []apim.ApimInstanceCollectionAssetsInner {
	switch strings.ToLower(typ) {
	case "mule3":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetTechnology(), "mule3")
		})
	case "mule4":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetTechnology(), "mule4")
		})
	case "flexgateway":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetTechnology(), "flexGateway")
		})
	default:
		return []apim.ApimInstanceCollectionAssetsInner{}
	}
}

func FilterApiByContracts(instances []apim.ApimInstanceCollectionAssetsInner, filterFlag string) []apim.ApimInstanceCollectionAssetsInner {
	switch strings.ToLower(filterFlag) {
	case "nonempty":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return api.GetActiveContractsCount() > 0
		})
	case "empty":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return api.GetActiveContractsCount() == 0
		})
	default:
		return []apim.ApimInstanceCollectionAssetsInner{}
	}
}

func FilterApiByStatus(instances []apim.ApimInstanceCollectionAssetsInner, statusFlag string) []apim.ApimInstanceCollectionAssetsInner {
	switch strings.ToLower(statusFlag) {
	case "unregistered":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetStatus(), "unregistered")
		})
	case "active":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetStatus(), "active")
		})
	case "inactive":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return strings.EqualFold(api.GetStatus(), "inactive")
		})
	default:
		return []apim.ApimInstanceCollectionAssetsInner{}
	}
}

func FilterApisByVisibility(instances []apim.ApimInstanceCollectionAssetsInner, visibilityFlag string) []apim.ApimInstanceCollectionAssetsInner {
	switch strings.ToLower(visibilityFlag) {
	case "private":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return !api.GetIsPublic()
		})
	case "public":
		return FilterApis(instances, func(api apim.ApimInstanceCollectionAssetsInnerApisInner) bool {
			return api.GetIsPublic()
		})
	default:
		return []apim.ApimInstanceCollectionAssetsInner{}
	}

}

func CountApis(instances []apim.ApimInstanceCollectionAssetsInner) int {
	total := 0
	for _, i := range instances {
		total += len(i.GetApis())
	}
	return total
}

func API2Map(results []apim.ApimInstanceCollectionAssetsInner) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	for _, r := range results {
		for _, api := range r.GetApis() {
			data = append(data, map[string]any{
				"Api ID":                 api.GetId(),
				"Api Name":               strings.TrimSpace(r.GetExchangeAssetName()),
				"Version":                api.GetProductVersion(),
				"Asset Version":          api.GetAssetVersion(),
				"Asset ID":               api.GetAssetId(),
				"Active Contracts Count": api.GetActiveContractsCount(),
				"Status":                 api.GetStatus(),
				"Deprecated":             api.GetDeprecated(),
				"Type":                   api.GetTechnology(),
				"Is Public":              api.GetIsPublic(),
				"Created At":             *api.Audit.GetCreated().Date,
				"Last Active Date":       api.GetLastActiveDate(),
			})
		}
	}
	order := []string{"Api ID", "Api Name", "Version", "Asset ID", "Asset Version", "Active Contracts Count", "Status", "Deprecated", "Type", "Is Public", "Created At", "Last Active Date"}

	return data, order
}

func PrintApisTable(results []apim.ApimInstanceCollectionAssetsInner) {
	data, order := API2Map(results)
	utils.PrintGenericTable(data, order)
}

func ExportApisTable(fileName string, results []apim.ApimInstanceCollectionAssetsInner) error {
	data, order := API2Map(results)
	return utils.ExportGenericCSV(fileName, data, order)
}

var listApiCmd = &cobra.Command{
	Use:   "list-apis",
	Short: "List API Manager instances",
	Long:  "List API Manager instances with optional filters. For example, you can list only instances without any API contracts.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flag values.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		filterContract, _ := cmd.Flags().GetString("filter-contract")
		filterType, _ := cmd.Flags().GetString("filter-type")
		filterStatus, _ := cmd.Flags().GetString("filter-status")
		filterVisiblity, _ := cmd.Flags().GetString("filter-visibility")
		adminToken, _ := cmd.Flags().GetString("token")
		exportFile, _ := cmd.Flags().GetString("out")
		// Retrieve the authenticated client.
		client, err := anypoint.GetInitializedClient(adminToken)
		if err != nil {
			utils.PrintError("Error retrieving client %v\n", err)
			return
		}
		// Validate organization and environment.
		orgID, envID, err = utils.ValidateOrgEnv(client, orgID, envID)
		if err != nil {
			utils.PrintError("Validation error: %v", err)
			return
		}
		//Get All APIs
		instances, err := client.GetAllApis(ctx, orgID, envID)
		if err != nil {
			utils.PrintError("Error retrieving API Manager instances: %v", err)
			return
		}
		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)
		//Display initial result
		if len(instances) == 0 {
			utils.PrintWarning("No API Manager instances Found.")
			return
		} else {
			utils.PrintInfo("Found %d APIs\n", CountApis(instances))
		}
		// Apply type filtering if specified.
		if !strings.EqualFold(filterType, "all") {
			instances = FilterApiByType(instances, filterType)
			utils.PrintInfo("After applying %s type filter, %d apis remain\n", filterType, CountApis(instances))
		}
		//filter apis on contracts
		if !strings.EqualFold(filterContract, "all") {
			instances = FilterApiByContracts(instances, strings.ToLower(filterContract))
			utils.PrintInfo("After applying %s contract filter, %d apis remain\n", filterContract, CountApis(instances))
		}
		//filter apis on status
		if !strings.EqualFold(filterStatus, "all") {
			instances = FilterApiByStatus(instances, filterStatus)
			utils.PrintInfo("After applying %s status filter, %d apis remain\n", filterStatus, CountApis(instances))
		}
		if !strings.EqualFold(filterVisiblity, "all") {
			instances = FilterApisByVisibility(instances, filterVisiblity)
			utils.PrintInfo("After applying %s visibility filter, %d apis remain\n", filterVisiblity, CountApis(instances))
		}
		if len(instances) == 0 {
			utils.PrintWarning("No API Manager instances match the specified criteria.")
			return
		}
		// If exportFile is provided, export to CSV.
		if exportFile != "" {
			err := ExportApisTable(exportFile, instances)
			if err != nil {
				utils.PrintError("Error exporting results to CSV: %v\n", err)
				return
			}
			utils.PrintSuccess("Results successfully exported to %s\n", exportFile)
		} else {
			// Otherwise, print a summary table.
			PrintApisTable(instances)
		}
	},
}

func init() {
	listApiCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	listApiCmd.Flags().String("env", "", "The Environment ID. If not provided, the id from the saved context will be loaded if present.")
	listApiCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")
	//filter flags
	listApiCmd.Flags().String("filter-contract", "all", "Filter: 'all' (default) or 'empty' or 'nonempty'")
	listApiCmd.Flags().String("filter-type", "all", "API type filter: 'all' (default), 'mule4', 'mule3' or 'flexGateway'")
	listApiCmd.Flags().String("filter-status", "all", "API status filter: 'all' (default), 'unregistered', 'active' or 'inactive'")
	listApiCmd.Flags().String("filter-visibility", "all", "API visibility filter: 'all' (default), 'private' or 'public'")
	//export flags
	listApiCmd.Flags().StringP("out", "o", "", "Export output to CSV file")
}
