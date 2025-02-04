package cmd

import (
	"fmt"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/config" // adjust the import path as needed
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to the Anypoint Platform",
	Long:  `Authenticate and establish a connection to the Anypoint Platform using your credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		clientId, _ := cmd.Flags().GetString("clientId")
		clientSecret, _ := cmd.Flags().GetString("clientSecret")
		controlPlane, _ := cmd.Flags().GetString("controlplane")

		// Validate control plane input.
		serverIndex := cplane2serverindex(controlPlane)
		if serverIndex == -1 {
			fmt.Println("Invalid control plane. Valid values are 'eu', 'us', or 'gov'.")
			return
		}

		// Create the client; this will obtain an access token and set its expiration.
		client, err := anypoint.NewClient(ctx, serverIndex, clientId, clientSecret)
		if err != nil {
			fmt.Printf("Error connecting to Anypoint: %v\n", err)
			return
		}

		// Persist configuration values using Viper.
		// In production, consider more secure storage for sensitive values.
		viper.Set("clientId", clientId)
		viper.Set("clientSecret", clientSecret)
		viper.Set("serverIndex", serverIndex)
		viper.Set("accessToken", client.AccessToken)
		// Persist the expiration time in RFC3339 format.
		viper.Set("expiresAt", client.ExpiresAt.Format(time.RFC3339))

		if err := config.SaveConfig(); err != nil {
			fmt.Printf("Warning: Unable to persist configuration: %v\n", err)
		}

		fmt.Printf("Successfully connected. Access token valid until %s.\n", client.ExpiresAt.Format(time.RFC1123))
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().StringP("clientId", "i", "", "Anypoint Platform connected app client id")
	connectCmd.Flags().StringP("clientSecret", "p", "", "Anypoint Platform connected app client secret")
	connectCmd.Flags().StringP("controlplane", "c", "eu", "Control plane to use (eu, us, gov)")
	connectCmd.MarkFlagRequired("clientId")
	connectCmd.MarkFlagRequired("clientSecret")
}

// cplane2serverindex converts control plane name to server index.
func cplane2serverindex(cplane string) int {
	if cplane == "eu" {
		return 1
	} else if cplane == "us" {
		return 0
	} else if cplane == "gov" {
		return 2
	}
	return -1 // Return -1 for invalid control plane
}
