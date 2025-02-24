package runtime

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
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

// PrintError prints an error message in red and bold to standard error.
func PrintError(format string, a ...interface{}) {
	errPrinter := color.New(color.FgRed, color.Bold).SprintfFunc()
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, errPrinter(msg))
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

// PrintGenericTable prints a table from a slice of map[string]interface{}.
// Each map represents a row and keys represent columns.
// An optional headerOrder slice can be provided to control column order.
// If headerOrder is empty, the union of keys is computed and sorted alphabetically.
func PrintGenericTable(data []map[string]interface{}, headerOrder []string) {
	if len(data) == 0 {
		fmt.Println("No data to display.")
		return
	}

	// If no header order is provided, compute the union of keys from all rows.
	if len(headerOrder) == 0 {
		headerMap := make(map[string]struct{})
		for _, row := range data {
			for key := range row {
				headerMap[key] = struct{}{}
			}
		}
		for key := range headerMap {
			headerOrder = append(headerOrder, key)
		}
		sort.Strings(headerOrder)
	}

	// Create a new tabwriter with appropriate settings.
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header row.
	fmt.Fprintln(w, strings.Join(headerOrder, "\t"))

	// Print a separator row.
	separator := make([]string, len(headerOrder))
	for i, h := range headerOrder {
		separator[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(w, strings.Join(separator, "\t"))

	// Print each row using the defined header order.
	for _, row := range data {
		var rowValues []string
		for _, key := range headerOrder {
			if val, ok := row[key]; ok {
				rowValues = append(rowValues, fmt.Sprintf("%v", val))
			} else {
				rowValues = append(rowValues, "")
			}
		}
		fmt.Fprintln(w, strings.Join(rowValues, "\t"))
	}

	w.Flush()
}

// ExportGenericCSV writes a slice of map[string]interface{} to a CSV file.
// The CSV file will contain a header row (either provided via headerOrder or computed)
// and one row per data map.
func ExportGenericCSV(fileName string, data []map[string]interface{}, headerOrder []string) error {
	// Open the file for writing (create or truncate)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", fileName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// If no header order is provided, compute the union of keys from the data.
	if len(headerOrder) == 0 {
		headerMap := make(map[string]struct{})
		for _, row := range data {
			for key := range row {
				headerMap[key] = struct{}{}
			}
		}
		for key := range headerMap {
			headerOrder = append(headerOrder, key)
		}
		sort.Strings(headerOrder)
	}

	// Write the header row.
	if err := writer.Write(headerOrder); err != nil {
		return fmt.Errorf("error writing header to CSV: %w", err)
	}

	// Write each row.
	for _, row := range data {
		record := make([]string, len(headerOrder))
		for i, key := range headerOrder {
			if val, ok := row[key]; ok {
				record[i] = fmt.Sprintf("%v", val)
			} else {
				record[i] = ""
			}
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing record to CSV: %w", err)
		}
	}

	return nil
}
