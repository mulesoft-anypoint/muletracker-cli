package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
)

// cplane2serverindex converts control plane name to server index.
func Cplane2serverindex(cplane string) int {
	if cplane == "eu" {
		return 1
	} else if cplane == "us" {
		return 0
	} else if cplane == "gov" {
		return 2
	}
	return -1 // Return -1 for invalid control plane
}

func Serverindex2cplane(index int) string {
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

// ExportGenericCSV writes a slice of map[string]any to a CSV file.
// The CSV file will contain a header row (either provided via headerOrder or computed)
// and one row per data map.
func ExportGenericCSV(fileName string, data []map[string]any, headerOrder []string) error {
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
