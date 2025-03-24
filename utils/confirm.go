package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// GetConfirmed prompts user for destructive operation confirmation
func GetConfirmed(warning string) error {
	color.New(color.Bold, color.FgRed).Printf("⚠️  %s [y/N] ", warning)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if response != "y" && response != "yes" {
		return fmt.Errorf("operation aborted")
	}
	return nil
}
