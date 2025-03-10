package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const configFileName = ".muletracker" // default file name without extension

// InitConfig initializes Viper to read the configuration file.
// If cfgFile is provided, that file is used; otherwise, it defaults to $HOME/.muletracker.yaml.
// If the configuration file doesn't exist, it is created.
func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		// Ensure the directory exists.
		dir := filepath.Dir(cfgFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory %s: %w", dir, err)
		}
		viper.SetConfigFile(cfgFile)
		// If there's no extension, assume YAML.
		if filepath.Ext(cfgFile) == "" {
			viper.SetConfigType("yaml")
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to find home directory: %w", err)
		}
		viper.SetConfigName(configFileName)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
	}

	// Set default values.
	viper.SetDefault("serverIndex", 0)
	viper.SetDefault("clientId", "")
	viper.SetDefault("clientSecret", "")
	viper.SetDefault("org", "")
	viper.SetDefault("env", "")
	// Add any other defaults as needed.

	err := viper.ReadInConfig()
	if err != nil {
		// Check if the error is because the file doesn't exist.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			var configPath string
			if cfgFile != "" {
				configPath = cfgFile
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("unable to find home directory: %w", err)
				}
				configPath = filepath.Join(home, configFileName+".yaml")
			}
			// Create the file using WriteConfigAs.
			if err := viper.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("could not create config file at %s: %w", configPath, err)
			}
			// Read the newly created config.
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("error reading config file after creation: %w", err)
			}
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}

// SaveConfig persists the current configuration to file.
// If a config file is already loaded (via viper.ConfigFileUsed()), it writes to it.
// Otherwise, it writes to the default $HOME/.muletracker.yaml.
func SaveConfig() error {
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		// File exists, update it.
		return viper.WriteConfig()
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath := filepath.Join(home, configFileName+".yaml")
		return viper.WriteConfigAs(configPath)
	}
}
