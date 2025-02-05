package anypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// App represents an application as returned by the ARMUI endpoint.
type App struct {
	ID     string `json:"id"`
	Target struct {
		Type string `json:"type"`
	} `json:"target"`
	Artifact struct {
		LastUpdateTime int64  `json:"lastUpdateTime"`
		CreateTime     *int64 `json:"createTime"`
		Name           string `json:"name"`
		FileName       string `json:"fileName"`
	} `json:"artifact"`
	MuleVersion struct {
		Version          string `json:"version"`
		UpdateId         string `json:"updateId"`
		LatestUpdateId   string `json:"latestUpdateId"`
		EndOfSupportDate int64  `json:"endOfSupportDate"`
	} `json:"muleVersion"`
	IsDeploymentWaiting bool   `json:"isDeploymentWaiting"`
	LastReportedStatus  string `json:"lastReportedStatus"`
	Details             struct {
		Domain string `json:"domain"`
	} `json:"details"`
}

// AppsResponse models the response from the applications endpoint.
type AppsResponse struct {
	Data  []App `json:"data"`
	Total int   `json:"total"`
}

// GetApps retrieves all applications for a given org and env.
func (c *Client) GetApps(ctx context.Context, orgID, envID string) ([]App, error) {
	host, err := c.getServerHost()
	if err != nil {
		return nil, err
	}

	url := host + "/armui/api/v1/applications"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers.
	req.Header.Set("x-anypnt-org-id", orgID)
	req.Header.Set("x-anypnt-env-id", envID)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

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
	return appsResp.Data, nil
}
