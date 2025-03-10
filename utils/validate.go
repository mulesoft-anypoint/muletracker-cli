package utils

import (
	"fmt"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
)

// ValidateOrgEnv validates that both org and env values are available.
// If the provided orgID or envID is empty, it loads them from the client.
// If the client also doesn't have a value, it returns an error.
// Otherwise, it updates the client with the provided flag values.
// Returns the final organization and environment values.
func ValidateOrgEnv(client *anypoint.Client, orgID, envID string) (string, string, error) {
	// Validate organization
	if client.IsOrgEmpty() && orgID == "" {
		return "", "", fmt.Errorf("organization ID not provided")
	}
	// Validate environment
	if client.IsEnvEmpty() && envID == "" {
		return "", "", fmt.Errorf("environment ID not provided")
	}

	// Load org and env from client if not provided by flags,
	// otherwise update the client with the provided flags.
	if orgID == "" {
		orgID = client.Org
	} else {
		client.SetOrg(orgID)
	}
	if envID == "" {
		envID = client.Env
	} else {
		client.SetEnv(envID)
	}
	return orgID, envID, nil
}

func ValidateOrg(client *anypoint.Client, orgID string) (string, error) {
	// Validate organization
	if client.IsOrgEmpty() && orgID == "" {
		return "", fmt.Errorf("organization ID not provided")
	}

	// Load org and env from client if not provided by flags,
	// otherwise update the client with the provided flags.
	if orgID == "" {
		orgID = client.Org
	} else {
		client.SetOrg(orgID)
	}
	return orgID, nil
}
