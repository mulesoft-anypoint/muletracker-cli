package cmd

import (
	"fmt"
	"os"

	"github.com/mulesoft-anypoint/muletracker-cli/cmd/apim"
	"github.com/mulesoft-anypoint/muletracker-cli/cmd/exchange"
	"github.com/mulesoft-anypoint/muletracker-cli/cmd/runtime"
	"github.com/mulesoft-anypoint/muletracker-cli/config"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var Version = "dev" // Set by build flags

var rootCmd = &cobra.Command{
	Use:     "MuleTracker",
	Short:   "MuleTracker CLI for MuleSoft Monitoring and Management",
	Version: Version,
	Long: `MuleTracker is a comprehensive CLI tool built in Go for managing and monitoring MuleSoft environments.
It provides a unified interface to perform operations across different facets of MuleSoft:

  • Connect:
    Authenticate against the Anypoint Platform using either a connected app token or an admin token.
    The configuration (credentials, selected org/env, tokens, etc.) is persisted in a configuration file.

  • Monitor:
    Retrieve detailed metrics such as last-called time and request counts for MuleSoft applications.
    Supports filtering, concurrent queries with rate limiting, and CSV export for aggregated results.

  • Exchange:
    Manage Exchange applications—list, create, or delete them—with filtering capabilities (e.g. display apps missing API contracts).

  • API Management:
    Handle API Manager operations by listing API Manager instances with dynamic filters (by type, status, etc.).

  • Runtime:
    Execute additional runtime-related operations for MuleSoft environments.

Configuration is managed via a file (default: $HOME/.muletracker.yaml), which you can override using the --config flag.
Each command is designed to provide colorful, aligned output and clear error messages for a user-friendly experience.

Use 'MuleTracker -h' to see available commands and options.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize configuration using our config package.
		if err := config.InitConfig(cfgFile); err != nil {
			utils.PrintError("Error initializing configuration: %v", err)
			os.Exit(1)
		}
		utils.PrintHighlightedMessage(fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()))
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to MuleTracker CLI. Use -h for help on available commands.")
	},
}

func init() {
	rootCmd.SetVersionTemplate(`{{printf "%s version %s\n" .Name .Version}}`)
	rootCmd.AddCommand(exchange.ExchangeCmd)
	rootCmd.AddCommand(runtime.RuntimeCmd)
	rootCmd.AddCommand(apim.ApimCmd)

	// Here you can add persistent flags and configuration settings.
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.muletracker.yaml)")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
