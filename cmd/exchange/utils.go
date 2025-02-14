package exchange

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mulesoft-anypoint/anypoint-client-go/org"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
)

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

// PrintClientInfo prints non-sensitive client information in a colorful format.
func PrintClientInfo(ctx context.Context, client *anypoint.Client) {
	var bg *org.MasterBGDetail
	var err error
	var env string
	if !client.IsOrgEmpty() {
		bg, err = client.GetBusinessGroup(ctx, client.Org)
		if err != nil {
			fmt.Printf("Error retrieving org: %v\n", err)
		}
		if !client.IsEnvEmpty() {
			for _, e := range bg.GetEnvironments() {
				if e.GetId() == client.Env {
					env = e.GetName()
					break
				}
			}
		}
	}

	data := map[string]interface{}{
		"* Control Plane":     strings.ToUpper(serverindex2cplane(client.ServerIndex)),
		"* Business Group Id": bg.GetName(),
		"* Environment Id":    env,
		"* Connected App":     client.ClientId,
		"* Token Expires At":  client.ExpiresAt.Format(time.RFC1123),
		// "InfluxDB ID":             client.InfluxDbId,
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
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
		if len(k) > maxKeyLength {
			maxKeyLength = len(k)
		}
	}
	// Sort keys alphabetically.
	sort.Strings(keys)

	// Define a divider line.
	divider := strings.Repeat("-", maxKeyLength+25)

	// Print the header.
	fmt.Println(headerColor(header))
	fmt.Println(divider)

	// Print each key/value pair.
	for _, key := range keys {
		val := data[key]
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
		fmt.Printf("%-*s: %s\n", maxKeyLength, keyColor(key), valueColor(formattedVal))
	}

	// Print the divider again.
	fmt.Println(divider)
}
