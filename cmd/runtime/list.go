package runtime

import (
	"fmt"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/spf13/cobra"
)

func Apps2Map(results []anypoint.App) ([]map[string]interface{}, []string) {
	data := make([]map[string]interface{}, 0)
	for _, r := range results {
		if anypoint.FilterRTF(r) {
			data = append(data, map[string]interface{}{
				"App Name":     r.Artifact.Name,
				"App Status":   r.Application.Status,
				"Mule Version": r.MuleVersion.Version,
				"Target":       "RTF",
			})
		} else if anypoint.FilterCH1(r) {
			data = append(data, map[string]interface{}{
				"App Name":     r.Artifact.Name,
				"App Status":   r.LastReportedStatus,
				"Mule Version": r.MuleVersion.Version,
				"Target":       "CLOUDHUB",
			})
		}
	}
	order := []string{"App ID", "App Name", "App Status", "Mule Version", "Target"}

	return data, order
}

// printAppsSummaryTable prints a condensed table of app monitoring results
// using tabwriter for alignment.
func printAppsSummaryTable(results []anypoint.App) {
	data, order := Apps2Map(results)
	PrintGenericTable(data, order)
}

func ExportAppsToCSV(fileName string, results []anypoint.App) error {
	data, order := Apps2Map(results)
	return ExportGenericCSV(fileName, data, order)
}

var listAppsCmd = &cobra.Command{
	Use:   "list",
	Short: "List Mule Applications",
	Long:  `List all or parts of Mule Applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flags.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		adminToken, _ := cmd.Flags().GetString("admin-token")
		exportFile, _ := cmd.Flags().GetString("out")
		// Retrieve the authenticated client.
		var client *anypoint.Client
		var err error
		if adminToken != "" {
			client, err = anypoint.GetClientFromContext(anypoint.WithSkipTokenExpiration())
			if err != nil {
				PrintError("Error retrieving client: %v\n", err)
				return
			}
			client.SetAdminAccessToken(adminToken)
		} else {
			client, err = anypoint.GetClientFromContext()
			if err != nil {
				PrintError("Error retrieving client: %v\n", err)
				return
			}
		}
		// Check that the required flags are provided.
		if (client.IsOrgEmpty() && orgID == "") || (client.IsEnvEmpty() && envID == "") {
			PrintError("Please provide --org, --env flags")
			return
		}

		//Load org and env if necessary
		if orgID == "" {
			orgID = client.Org
		} else {
			client.SetOrg(orgID)
		}
		if envID == "" {
			envID = client.Env
		} else {
			client.SetEnv(envID)
		}
		//Get All exchange client apps
		apps, err := client.GetApps(ctx, orgID, envID, []anypoint.AppFilter{}...)
		if err != nil {
			PrintError("Error retrieving apps: %v", err)
		}
		// Display the client info in a colorful way.
		PrintClientInfo(ctx, client)

		// If export flag is provided, export results to CSV.
		if exportFile != "" {
			err := ExportAppsToCSV(exportFile, apps)
			if err != nil {
				PrintError("Error exporting results to CSV: %v\n", err)
				return
			}
			fmt.Printf("\nResults successfully exported to %s\n", exportFile)
		} else {
			// Otherwise, print a summary table.
			printAppsSummaryTable(apps)
		}
	},
}

func init() {
	listAppsCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	listAppsCmd.Flags().String("env", "", "The Environment ID. If not provided, the id from the saved context will be loaded if present")
	listAppsCmd.Flags().StringP("admin-token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")

	// export flags
	listAppsCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
}
