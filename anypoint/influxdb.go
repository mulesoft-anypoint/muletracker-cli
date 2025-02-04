package anypoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Servers mapping for building the base URL dynamically.
var monitoringServers = []string{
	"https://anypoint.mulesoft.com/monitoring/api/visualizer/api/datasources/proxy/88/query",
	"https://eu1.anypoint.mulesoft.com/monitoring/api/visualizer/api/datasources/proxy/88/query",
	"https://gov.anypoint.mulesoft.com/monitoring/api/visualizer/api/datasources/proxy/88/query",
}

// getQueryBaseURL returns the monitoring query URL for the client’s server index.
func (c *Client) getQueryBaseURL() (string, error) {
	if c.ServerIndex < 0 || c.ServerIndex >= len(monitoringServers) {
		return "", errors.New("invalid server index")
	}
	return monitoringServers[c.ServerIndex], nil
}

// QueryParams holds the parameters for querying the InfluxDB API.
type QueryParams struct {
	// For our purposes, these are used to build the query.
	OrgID string
	EnvID string
	AppID string
	Query string
}

// InfluxDBResponse represents the structure of the InfluxDB API response.
type InfluxDBResponse struct {
	Results []struct {
		StatementID int `json:"statement_id"`
		Series      []struct {
			Name    string          `json:"name"`
			Tags    interface{}     `json:"tags"`
			Columns []string        `json:"columns"`
			Values  [][]interface{} `json:"values"`
		} `json:"series"`
	} `json:"results"`
}

// queryInfluxDB performs the query against the monitoring endpoint and returns the parsed response.
func (c *Client) queryInfluxDB(ctx context.Context, params QueryParams) (*InfluxDBResponse, error) {
	baseURL, err := c.getQueryBaseURL()
	if err != nil {
		return nil, err
	}

	// Build URL query parameters.
	q := url.Values{}
	// Hardcoded database value from your example.
	q.Add("db", `"dias"`)
	q.Add("q", params.Query)
	q.Add("epoch", "ms")

	// Construct the full URL.
	fullURL := fmt.Sprintf("%s?%s", baseURL, q.Encode())

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add the Bearer token from the client's accessToken.
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Execute the HTTP request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var influxResp InfluxDBResponse
	if err := json.Unmarshal(body, &influxResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &influxResp, nil
}
