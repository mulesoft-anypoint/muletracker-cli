package main

import (
	"log"

	"github.com/mulesoft-anypoint/muletracker-cli/cmd"
	"github.com/mulesoft-anypoint/muletracker-cli/config"
)

func main() {
	// Initialize configuration using Viper.
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Error initializing config: %v", err)
	}

	// Run the CLI.
	cmd.Execute()
}
