package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/config" // adjust the import path as needed
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// environmentsCmd represents the environment command
var environmentsCmd = &cobra.Command{
	Use:   "environment",
	Short: "Get Environment Details",
	Long:  `Retrieve and display Environment details for a specific Business Group, then allow selection of one to persist.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		businessGroupID, _ := cmd.Flags().GetString("org")
		if businessGroupID == "" {
			fmt.Println("Please provide a business group ID using the --org flag.")
			return
		}

		// Persist the Business Group (Org) ID.
		viper.Set("org", businessGroupID)
		if err := config.SaveConfig(); err != nil {
			fmt.Printf("Warning: Unable to persist organization configuration: %v\n", err)
		}

		// Retrieve the authenticated client.
		client, err := anypoint.GetClientFromContext()
		if err != nil {
			fmt.Printf("Error retrieving client: %v\n", err)
			return
		}

		// Retrieve environments for the provided business group.
		environments, err := client.GetEnvironments(ctx, businessGroupID)
		if err != nil {
			fmt.Printf("Error retrieving environments: %v\n", err)
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
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		input = strings.TrimSpace(input)
		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(environments) {
			fmt.Println("Invalid selection.")
			return
		}

		selectedEnv := environments[selection-1]
		// Persist the selected Environment ID.
		viper.Set("env", selectedEnv.GetId())
		if err := config.SaveConfig(); err != nil {
			fmt.Printf("Warning: Unable to persist environment configuration: %v\n", err)
		}

		fmt.Printf("Selected environment: %s (ID: %s)\n", selectedEnv.GetName(), selectedEnv.GetId())
	},
}

func init() {
	rootCmd.AddCommand(environmentsCmd)
	environmentsCmd.Flags().StringP("org", "o", "", "Business Group ID")
	environmentsCmd.MarkFlagRequired("org")
}
