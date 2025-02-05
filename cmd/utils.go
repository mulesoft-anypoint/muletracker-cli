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
	data := map[string]interface{}{
		"Connected App Client ID": client.ClientId,
		"Server Index":            client.ServerIndex,
		"Token Expires At":        client.ExpiresAt.Format(time.RFC1123),
		"InfluxDB ID":             client.InfluxDbId,
	}

	PrintSimpleResults("Client Information:", data)
}

// PrintSimpleResults prints a header and key/value pairs in a simple, aligned style.
func PrintSimpleResults(header string, data map[string]interface{}) {
	// Define color functions.
	headerColor := color.New(color.FgGreen, color.Bold).SprintFunc()
	keyColor := color.New(color.FgYellow).SprintFunc()
	valueColor := color.New(color.FgWhite).SprintFunc()

	// Determine the maximum key width for alignment.
	maxKeyLength := 0
	for k := range data {
		if len(k) > maxKeyLength {
			maxKeyLength = len(k)
		}
	}

	// Define a divider line.
	divider := strings.Repeat("-", maxKeyLength+25)

	// Print the header.
	fmt.Println(headerColor(header))
	fmt.Println(divider)

	// Print each key/value pair.
	for key, val := range data {
		// Format time values specially.
		var formattedVal string
		switch t := val.(type) {
		case time.Time:
			if t.IsZero() {
				formattedVal = "No data available"
			} else {
				formattedVal = t.Format(time.RFC1123)
			}
		default:
			formattedVal = fmt.Sprintf("%v", val)
		}

		// Left-align the key using the maximum width.
		fmt.Printf("%s: %s\n", keyColor(fmt.Sprintf("%-*s", maxKeyLength, key)), valueColor(formattedVal))
	}

	// Print the divider again.
	fmt.Println(divider)
}
