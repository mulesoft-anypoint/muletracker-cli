package exchange

import (
	"fmt"
	"strconv"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/spf13/cobra"
)

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
		adminToken, _ := cmd.Flags().GetString("adminToken")
		// Validate params
		if appIDStr == "" {
			fmt.Println("Error: Application ID is required to delete an Exchange application.")
			return
		}
		appID, err := strconv.Atoi(appIDStr)
		if err != nil {
			fmt.Println("Error: Application ID is unvalid.")
			return
		}
		// Retrieve the authenticated client.
		var client *anypoint.Client
		if adminToken != "" {
			client, err = anypoint.GetClientFromContext(anypoint.WithSkipTokenExpiration())
			if err != nil {
				fmt.Printf("Error retrieving client: %v\n", err)
				return
			}
			client.SetAdminAccessToken(adminToken)
		} else {
			client, err = anypoint.GetClientFromContext()
			if err != nil {
				fmt.Printf("Error retrieving client: %v\n", err)
				return
			}
		}
		//Read Org ID
		if client.IsOrgEmpty() && orgID == "" {
			fmt.Println("Please provide --org flag")
			return
		}
		if orgID == "" {
			orgID = client.Org
		} else {
			client.SetOrg(orgID)
		}
		//Perform Delete Action
		if err := client.DeleteExchangeClientApp(ctx, orgID, int32(appID)); err != nil {
			fmt.Printf("Error deleting the application client %v\n", err)
			return
		}
		fmt.Printf("Deleted Exchange application with ID: %s\n", appIDStr)
	},
}

func init() {
	// Define flags for the delete command.
	deleteClientAppCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	deleteClientAppCmd.Flags().String("id", "", "ID of the Exchange application to delete (required)")
	deleteClientAppCmd.Flags().StringP("adminToken", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all the org's client applications")
	//required flags
	deleteClientAppCmd.MarkFlagRequired("id")
}
