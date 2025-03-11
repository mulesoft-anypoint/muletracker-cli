package apim

import (
	"fmt"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete-api",
	Short: "Delete an API Manager instance",
	Long:  "Delete an API Manager instance by specifying its ID. This command removes the instance from the API Management system.",
	Run: func(cmd *cobra.Command, args []string) {
		// Retrieve required flag.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		adminToken, _ := cmd.Flags().GetString("token")
		instanceID, _ := cmd.Flags().GetString("id")
		if instanceID == "" {
			utils.PrintError("Error: Instance ID is required to delete an API Manager instance.")
			return
		}

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

		// Call the delete function (stubbed for now).
		err = client.DeleteApi(cmd.Context(), orgID, envID, instanceID)
		if err != nil {
			utils.PrintError("Error deleting API Manager instance %s: %v", instanceID, err)
			return
		}

		fmt.Printf("Successfully deleted API Manager instance: %s\n", instanceID)
	},
}

func init() {
	deleteCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	deleteCmd.Flags().String("env", "", "The Environment ID. If not provided, the id from the saved context will be loaded if present.")
	deleteCmd.Flags().String("id", "", "ID of the API Manager instance to delete")
	deleteCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")
}
