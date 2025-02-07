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
		Type    string `json:"type"`
		Subtype string `json:"subtype,omitempty"`
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
	Application         struct {
		Status string `json:"status"`
	} `json:"application",omitempty`
	Details struct {
		Domain string `json:"domain"`
	} `json:"details"`
}

func (a App) GetType() string {
	if a.Target.Type == "MC" {
		return a.Target.Subtype
	}
	return a.Target.Type
}

// AppsResponse models the response from the applications endpoint.
type AppsResponse struct {
	Data  []App `json:"data"`
	Total int   `json:"total"`
}

// AppFilter defines a function type for filtering apps.
type AppFilter func(app App) bool

// GetApps retrieves all applications for a given org and env.
func (c *Client) GetApps(ctx context.Context, orgID, envID string, filters ...AppFilter) ([]App, error) {
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

	apps := appsResp.Data
	if len(filters) > 0 {
		apps = FilterApps(apps, filters...)
	}
	return apps, nil
}

// FilterApps returns a slice of apps that match all provided filters.
func FilterApps(apps []App, filters ...AppFilter) []App {
	var filtered []App
	for _, app := range apps {
		match := true
		for _, filter := range filters {
			if !filter(app) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// FilterCloudhub returns true if an app is deployed to CloudHub.
func FilterCloudhub(app App) bool {
	return app.Target.Type == "CLOUDHUB"
}

// FilterRTF returns true if an app is deployed to RTF (runtime fabrics).
func FilterRTF(app App) bool {
	return app.Target.Type == "MC" && app.Target.Subtype == "runtime-fabrics"
}

// FilterRunning returns true if an app is running.
func FilterRunning(app App) bool {
	// Check for CloudHub apps.
	if FilterCloudhub(app) {
		return app.LastReportedStatus != "STARTED"
	}
	// Check for RTF apps.
	if FilterRTF(app) {
		return app.Application.Status != "RUNNING"
	}
	// For other types, do not filter them out.
	return true
}
