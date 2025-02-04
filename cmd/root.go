package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "MuleTracket",
	Short: "MuleTracket monitors MuleSoft app activity",
	Long: `MuleTracket is a CLI tool built in Go to monitor MuleSoft applications.
It allows you to connect to the Anypoint Platform, navigate through Business Groups
and Environments, and analyze application usage such as last call time and request counts.`,
	// You can add a Run function if you want default behavior:
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to MuleTracket CLI. Use -h for help on available commands.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you can add persistent flags and configuration settings.
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.muletracker.yaml)")
}
