package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mulesoft-anypoint/anypoint-client-go/org"
	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
)

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

// ExportResultsToCSV writes the provided AppResult slice to a CSV file.
// The CSV file will contain a header row and one row per result.
func ExportResultsToCSV(fileName string, results []AppResult) error {
	// Open the file for writing (create or truncate)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", fileName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row.
	header := []string{"App ID", "Last Called", "Request Count", "LC Window", "RC Window"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing header to CSV: %w", err)
	}

	// Write each row.
	for _, res := range results {
		var lastCalled string
		if res.LastCalled.IsZero() {
			lastCalled = "No data"
		} else {
			// Format time in a friendly format.
			lastCalled = res.LastCalled.Format(time.RFC1123)
		}
		record := []string{
			res.AppID,
			lastCalled,
			fmt.Sprintf("%d", res.RequestCount),
			res.LCWindow,
			res.RCWindow,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing record for app %s: %w", res.AppID, err)
		}
	}

	return nil
}
