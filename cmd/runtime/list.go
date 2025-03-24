package runtime

import (
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func parseMuleVersionDate(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}
	t := time.UnixMilli(timestamp)
	return t.Format("Jan 2, 2006")
}

func Apps2Map(results []anypoint.App) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	for _, r := range results {
		if anypoint.FilterRTF(r) {
			data = append(data, map[string]any{
				"App Name":         r.Artifact.Name,
				"App Status":       r.Application.Status,
				"Mule Version":     r.MuleVersion.Version,
				"Mule Version EOL": parseMuleVersionDate(r.MuleVersion.EndOfSupportDate),
				"Target":           "RTF",
			})
		} else if anypoint.FilterCH1(r) {
			data = append(data, map[string]any{
				"App Name":         r.Artifact.Name,
				"App Status":       r.LastReportedStatus,
				"Mule Version":     r.MuleVersion.Version,
				"Mule Version EOL": parseMuleVersionDate(r.MuleVersion.EndOfSupportDate),
				"Target":           "CLOUDHUB",
			})
		}
	}
	order := []string{"App Name", "App Status", "Target", "Mule Version", "Mule Version EOL"}

	return data, order
}

// printAppsSummaryTable prints a condensed table of app monitoring results
// using tabwriter for alignment.
func printListAppsTable(results []anypoint.App) {
	data, order := Apps2Map(results)
	utils.PrintGenericTable(data, order)
}

func ExportAppsToCSV(fileName string, results []anypoint.App) error {
	data, order := Apps2Map(results)
	return utils.ExportGenericCSV(fileName, data, order)
}

var listAppsCmd = &cobra.Command{
	Use:   "list-apps",
	Short: "List Mule Applications",
	Long:  `List all or parts of Mule Applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flags.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		adminToken, _ := cmd.Flags().GetString("token")
		exportFile, _ := cmd.Flags().GetString("out")
		// Retrieve the authenticated client.
		client, err := anypoint.GetInitializedClient(adminToken)
		if err != nil {
			utils.PrintError("Error retrieving client %v\n", err)
			return
		}
		// 2. Validate organization and environment.
		orgID, envID, err = utils.ValidateOrgEnv(client, orgID, envID)
		if err != nil {
			utils.PrintError("Validation error: %v", err)
			return
		}
		//Get All exchange client apps
		apps, err := client.GetApps(ctx, orgID, envID, []anypoint.AppFilter{}...)
		if err != nil {
			utils.PrintError("Error retrieving apps: %v", err)
		}
		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)

		// If export flag is provided, export results to CSV.
		if exportFile != "" {
			err := ExportAppsToCSV(exportFile, apps)
			if err != nil {
				utils.PrintError("Error exporting results to CSV: %v\n", err)
				return
			}
			utils.PrintSuccess("Results successfully exported to %s", exportFile)
		} else {
			// Otherwise, print a summary table.
			printListAppsTable(apps)
		}
	},
}

func init() {
	listAppsCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	listAppsCmd.Flags().String("env", "", "The Environment ID. If not provided, the id from the saved context will be loaded if present.")
	listAppsCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")

	// export flags
	listAppsCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
}
