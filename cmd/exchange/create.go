package exchange

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an Exchange application",
	Long:  `Create a new Exchange application. Provide necessary metadata such as name, description, and other settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Retrieve flags.
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		// Validate required fields.
		if name == "" {
			fmt.Println("Error: Application name is required.")
			return
		}

		// TODO: Call your Exchange client library to create an app.
		fmt.Printf("Creating Exchange application: %s\nDescription: %s\n", name, description)
	},
}

func init() {
	// Define flags for creating an Exchange app.
	createCmd.Flags().String("name", "", "Name of the Exchange application (required)")
	createCmd.Flags().String("description", "", "Description for the Exchange application")
	createCmd.MarkFlagRequired("name")
}
