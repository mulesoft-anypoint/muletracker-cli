package cmd

import (
	"fmt"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
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

		// Attempt to get clientId and clientSecret from flags;
		// if not provided, read them from persisted configuration.
		clientId, _ := cmd.Flags().GetString("clientId")
		if clientId == "" {
			clientId = viper.GetString("clientId")
		}

		clientSecret, _ := cmd.Flags().GetString("clientSecret")
		if clientSecret == "" {
			clientSecret = viper.GetString("clientSecret")
		}

		// Attempt to get controlplane from flag; if not provided, read from configuration.
		controlPlane, _ := cmd.Flags().GetString("controlplane")
		if controlPlane == "" {
			controlPlane = viper.GetString("controlplane")
		}
		// If still empty, default to "us"
		if controlPlane == "" {
			controlPlane = "us"
		}

		// Validate that we have credentials.
		if clientId == "" || clientSecret == "" {
			fmt.Println("clientId and clientSecret are required. Please provide them via flags or ensure they are persisted in configuration.")
			return
		}

		// Validate control plane and determine the server index.
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

		// Display the client info in a colorful way.
		PrintClientInfo(ctx, client)

		fmt.Printf("Successfully connected. Access token valid until %s.\n", client.ExpiresAt.Format(time.RFC1123))
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().StringP("clientId", "i", "", "Anypoint Platform connected app client id")
	connectCmd.Flags().StringP("clientSecret", "s", "", "Anypoint Platform connected app client secret")
	connectCmd.Flags().StringP("controlplane", "c", "", "Control plane to use (eu, us, gov)")
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

func serverindex2cplane(index int) string {
	switch index {
	case 0:
		return "us"
	case 1:
		return "eu"
	case 2:
		return "gov"
	default:
		return "unknown"
	}
}
