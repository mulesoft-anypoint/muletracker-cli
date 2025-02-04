package cmd

import (
	"fmt"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/spf13/cobra"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor MuleSoft App Activity",
	Long: `Analyze application behavior by checking when the app was last called
and how many requests it received over a specified time window.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Retrieve the context from the command.
		ctx := cmd.Context()

		// Retrieve required flags.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		appID, _ := cmd.Flags().GetString("app")
		lcWindow, _ := cmd.Flags().GetString("last-called-window")
		rcWindow, _ := cmd.Flags().GetString("request-count-window")

		// Check that the required flags are provided.
		if orgID == "" || envID == "" || appID == "" {
			fmt.Println("Please provide --org, --env, and --app flags")
			return
		}

		// Retrieve the previously connected client from context.
		client, err := anypoint.GetClientFromContext()
		if err != nil {
			fmt.Printf("Error retrieving client: %v\n", err)
			return
		}

		// Get the last-called time using the specified time window.
		lastCalled, err := client.GetLastCalledTime(ctx, orgID, envID, appID, lcWindow)
		if err != nil {
			fmt.Printf("Error retrieving last called time: %v\n", err)
			return
		}

		// Get the request count using the specified time window.
		requestCount, err := client.GetRequestCount(ctx, orgID, envID, appID, rcWindow)
		if err != nil {
			fmt.Printf("Error retrieving request count: %v\n", err)
			return
		}

		// Output the results.
		fmt.Printf("App ID: %s\n", appID)
		fmt.Printf("Last Called Time (over last %s): %s\n", lcWindow, lastCalled.Format(time.RFC1123))
		fmt.Printf("Request Count (over last %s): %d\n", rcWindow, requestCount)
	},
}

func init() {
	// Add the monitor command to the root command.
	rootCmd.AddCommand(monitorCmd)

	// Define flags for organization, environment, and application IDs.
	monitorCmd.Flags().String("org", "o", "Organization ID")
	monitorCmd.Flags().String("env", "e", "Environment ID")
	monitorCmd.Flags().String("app", "a", "Application ID to monitor")

	// Define flags for specifying the time window for queries.
	monitorCmd.Flags().String("last-called-window", "15m", "Time window for last-called query (e.g., 15m, 1h, 24h)")
	monitorCmd.Flags().String("request-count-window", "24h", "Time window for request count query (e.g., 24h, 3d)")

	// Mark the required flags.
	monitorCmd.MarkFlagRequired("org")
	monitorCmd.MarkFlagRequired("env")
	monitorCmd.MarkFlagRequired("app")
}
