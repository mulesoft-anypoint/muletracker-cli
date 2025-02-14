package exchange

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an Exchange application",
	Long:  `Delete an Exchange application. You must provide the application ID or name.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Retrieve flags.
		appID, _ := cmd.Flags().GetString("id")
		if appID == "" {
			fmt.Println("Error: Application ID is required to delete an Exchange application.")
			return
		}

		// TODO: Call your Exchange client library to delete the app.
		fmt.Printf("Deleting Exchange application with ID: %s\n", appID)
	},
}

func init() {
	// Define flags for the delete command.
	deleteCmd.Flags().String("id", "", "ID of the Exchange application to delete (required)")
	deleteCmd.MarkFlagRequired("id")
}
