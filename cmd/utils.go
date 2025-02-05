package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
)

// PrintClientInfo prints non-sensitive client information in a colorful format.
func PrintClientInfo(client *anypoint.Client) {
	// Create color functions.
	header := color.New(color.FgGreen, color.Bold).SprintFunc()
	key := color.New(color.FgYellow).SprintFunc()
	value := color.New(color.FgWhite).SprintFunc()
	separator := color.New(color.FgGreen).SprintFunc()

	// Define a separator line.
	sepLine := separator(strings.Repeat("=", 50))

	// Print a header.
	fmt.Println(sepLine)
	fmt.Println(header("Client Information:"))
	fmt.Println(sepLine)
	// Print non-sensitive fields.
	fmt.Printf("%s: %s\n", key("Connected App Client ID"), value(client.ClientId))
	fmt.Printf("%s: %s\n", key("Server Index"), serverindex2cplane(client.ServerIndex))
	fmt.Printf("%s: %s\n", key("Token Expires At"), value(client.ExpiresAt.Format(time.RFC1123)))
	fmt.Printf("%s: %d\n", key("InfluxDB ID"), client.InfluxDbId)
	fmt.Println(sepLine)
}
