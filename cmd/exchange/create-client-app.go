package exchange

import (
	"strings"

	"github.com/mulesoft-anypoint/anypoint-client-go/exchange_client_apps"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

func ExchangeClientApps2Map(apps []exchange_client_apps.ClientApp) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	for _, a := range apps {
		data = append(data, map[string]any{
			"App ID":        a.GetId(),
			"App Name":      a.GetName(),
			"Client ID":     a.GetClientId(),
			"Client Secret": a.GetClientSecret(),
		})
	}
	order := []string{"App ID", "App Name", "Client ID", "Client Secret"}
	return data, order
}

func PrintClientApps(apps []exchange_client_apps.ClientApp) {
	data, order := ExchangeClientApps2Map(apps)
	utils.PrintGenericTable(data, order)
}

var createClientAppCmd = &cobra.Command{
	Use:   "create-client-app",
	Short: "Create an Exchange client application",
	Long:  `Create a new Exchange client application. Provide necessary metadata such as name, description, and other settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Retrieve flags.
		orgID, _ := cmd.Flags().GetString("org")
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		redirectUri, _ := cmd.Flags().GetString("redirect-uris")
		url, _ := cmd.Flags().GetString("url")
		grantTypes, _ := cmd.Flags().GetString("grant-types")
		adminToken, _ := cmd.Flags().GetString("token")
		// Validate required fields.
		if name == "" {
			utils.PrintError("Error: Application name is required.")
			return
		}
		// Retrieve the authenticated client.
		client, err := anypoint.GetInitializedClient(adminToken)
		if err != nil {
			utils.PrintError("Error retrieving client %v\n", err)
			return
		}
		//Read Org ID
		orgID, err = utils.ValidateOrg(client, orgID)
		if err != nil {
			utils.PrintError("Validation error: %v", err)
			return
		}
		//Create the Client App
		app, err := client.PostExchangeClientApp(ctx, orgID, name, description, url, strings.Split(grantTypes, ","), strings.Split(redirectUri, ","))
		if err != nil {
			utils.PrintError("Error creating the client app %v\n", err)
			return
		}

		PrintClientApps([]exchange_client_apps.ClientApp{*app})
	},
}

func init() {
	// Define flags for creating an Exchange app.
	createClientAppCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	createClientAppCmd.Flags().String("name", "", "Name of the Exchange application (required)")
	createClientAppCmd.Flags().String("description", "", "Description for the Exchange application (optional).")
	createClientAppCmd.Flags().String("redirect-uris", "", "OAuth 2.0 redirect URIs separated by comma (optional).")
	createClientAppCmd.Flags().String("url", "", "The application URL (optional).")
	createClientAppCmd.Flags().String("grant-types", "", "The application grant types separated by comma (optional).")
	createClientAppCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all the org's client applications")
	//Required name
	createClientAppCmd.MarkFlagRequired("name")
}
