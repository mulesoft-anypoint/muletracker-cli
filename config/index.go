package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const configFileName = ".muletracker" // without extension

// InitConfig sets up Viper to read in the configuration file.
func InitConfig() error {
	// Find home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Tell viper the name of the config file (without extension)
	viper.SetConfigName(configFileName)
	// Add the home directory as the first search path.
	viper.AddConfigPath(home)

	// Optionally, you can set defaults.
	viper.SetDefault("serverIndex", 0)
	viper.SetDefault("clientId", "")
	viper.SetDefault("clientSecret", "")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// If the error is because the file doesn't exist, you might want to create one.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Create a new file with default values.
			configPath := filepath.Join(home, configFileName+".yaml")
			if err := viper.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("could not create config file: %w", err)
			}
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}

// SaveConfig persists the current configuration to file.
func SaveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, configFileName+".yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return err
	}
	return nil
}
