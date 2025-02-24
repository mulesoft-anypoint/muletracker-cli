package anypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetApps retrieves all applications for a given org and env.
func (c *Client) GetApps(ctx context.Context, orgID, envID string, filters ...AppFilter) ([]App, error) {
	host, err := c.getServerHost()
	if err != nil {
		return nil, err
	}
	token := c.getEffectiveToken()
	url := host + "/armui/api/v1/applications"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers.
	req.Header.Set("x-anypnt-org-id", orgID)
	req.Header.Set("x-anypnt-env-id", envID)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-OK status %d: %s", resp.StatusCode, string(body))
	}

	var appsResp AppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&appsResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	apps := appsResp.Data
	if len(filters) > 0 {
		apps = FilterApps(apps, filters...)
	}
	return apps, nil
}
