package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

// environmentsCmd represents the environment command
var environmentsCmd = &cobra.Command{
	Use:   "environment",
	Short: "Get Environment Details",
	Long:  `Retrieve and display Environment details for a specific Business Group, then allow selection of one to persist.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Parse Flags
		orgID, _ := cmd.Flags().GetString("org")
		adminToken, _ := cmd.Flags().GetString("token")
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
		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)

		// Retrieve environments for the provided business group.
		environments, err := client.GetEnvironments(ctx, orgID)
		if err != nil {
			utils.PrintError("Error retrieving environments: %v\n", err)
			return
		}

		if len(environments) == 0 {
			fmt.Println("No environments found.")
			return
		}

		// List the available environments.
		fmt.Println("Environments:")
		for idx, env := range environments {
			fmt.Printf("%d) %s (ID: %s)\n", idx+1, env.GetName(), env.GetId())
		}

		// Prompt the user to select an environment.
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Select environment number to use: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			utils.PrintError("Error reading input: %v\n", err)
			return
		}
		input = strings.TrimSpace(input)
		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(environments) {
			utils.PrintError("Error: Invalid selection.")
			return
		}

		selectedEnv := environments[selection-1]

		client.SetOrg(orgID)
		client.SetEnv(selectedEnv.GetId())

		utils.PrintSuccess("Selected environment: %s (ID: %s)", selectedEnv.GetName(), selectedEnv.GetId())
	},
}

func init() {
	rootCmd.AddCommand(environmentsCmd)
	environmentsCmd.Flags().String("org", "", "The Business Group ID (required).")
	environmentsCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")

}
