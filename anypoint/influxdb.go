package anypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// A path template for the monitoring API. The "%s" will be replaced with the InfluxDB ID.
var influxDBPathTemplate = "/monitoring/api/visualizer/api/datasources/proxy/%s/query"

// QueryParams holds the parameters for querying the InfluxDB API.
type QueryParams struct {
	// For our purposes, these are used to build the query.
	OrgID      string
	EnvID      string
	AppID      string
	Query      string
	InfluxDBId int
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

// BootDataResponseMinimal models just the portion of the bootdata JSON we need.
type BootDataResponseMinimal struct {
	Settings struct {
		Datasources struct {
			Influxdb struct {
				ID int `json:"id"`
			} `json:"influxdb"`
		} `json:"datasources"`
	} `json:"Settings"`
}

// queryInfluxDB performs the query against the monitoring endpoint and returns the parsed response.
func (c *Client) queryInfluxDB(ctx context.Context, params QueryParams) (*InfluxDBResponse, error) {
	// Get the host URL based on the client’s serverIndex.
	host, err := c.getServerHost()
	if err != nil {
		return nil, err
	}

	// Ensure that an InfluxDB ID is provided; otherwise, use a default or return an error.
	influxID := params.InfluxDBId
	if influxID == 0 {
		return nil, fmt.Errorf("influxDB ID not provided")
	}

	// Construct the path by substituting the influxID into the path template.
	path := fmt.Sprintf(influxDBPathTemplate, strconv.Itoa(influxID))

	// Combine the host and path.
	baseURL := host + path

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

	if resp.StatusCode != http.StatusOK {
		// Read the body to provide additional error details.
		body, _ := io.ReadAll(resp.Body)
		// Debug log: print the raw response body (remove in production)
		fmt.Printf("Raw response: %s\n", string(body))
		return nil, fmt.Errorf("received non-OK HTTP status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var influxResp InfluxDBResponse
	if err := json.Unmarshal(body, &influxResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &influxResp, nil
}

// GetInfluxDBID calls the bootdata endpoint and extracts the influxdb id.
func (c *Client) GetInfluxDBID(ctx context.Context) (int, error) {
	// Obtain the host using your helper (getMonitoringHost)
	host, err := c.getServerHost()
	if err != nil {
		return 0, err
	}
	bootDataURL := host + "/monitoring/api/visualizer/api/bootdata"
	token := c.getEffectiveToken()

	// Create the GET request.
	req, err := http.NewRequestWithContext(ctx, "GET", bootDataURL, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating bootdata request: %w", err)
	}

	// Set the Authorization header.
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute the request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error executing bootdata request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Debug log: print the raw response body (remove in production)
		fmt.Printf("Raw response: %s\n", string(body))
		return 0, fmt.Errorf("received non-OK HTTP status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading bootdata response: %w", err)
	}

	// Unmarshal only the required fields.
	var bootData BootDataResponseMinimal
	if err := json.Unmarshal(body, &bootData); err != nil {
		return 0, fmt.Errorf("error unmarshaling bootdata response: %w", err)
	}

	c.InfluxDbId = bootData.Settings.Datasources.Influxdb.ID
	// Return the influxdb id.
	return c.InfluxDbId, nil
}

// GetLastCalledTime fetches the last time the given app was called.
// It uses a query that calculates the 75th percentile of the avg_request_count
// over the specified time window. It returns the timestamp of the latest data point.
// The timeWindow parameter is a string (e.g. "15m", "24h", "3d") to define the lookback period.
func (c *Client) GetLastCalledTime(ctx context.Context, orgID, envID string, app App, timeWindow string) (time.Time, error) {
	templateCH1 := `SELECT percentile("avg_request_count", 75) FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`
	templateRTF := `SELECT percentile("avg_request_count", 75) FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "cluster_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`
	var query string

	if FilterCH1(app) {
		query = fmt.Sprintf(
			templateCH1,
			orgID, envID, app.Details.Domain, timeWindow,
		)
	} else if FilterRTF(app) {
		query = fmt.Sprintf(
			templateRTF,
			orgID, envID, app.Target.ID, app.Artifact.Name, timeWindow,
		)
	} else {
		fmt.Printf("Unsupported app target: %v\n", app)
		return time.Time{}, fmt.Errorf("unsupported app type: %s", app.Target.Type)
	}

	params := QueryParams{
		OrgID:      orgID,
		EnvID:      envID,
		AppID:      app.ID,
		Query:      query,
		InfluxDBId: c.InfluxDbId,
	}

	resp, err := c.queryInfluxDB(ctx, params)
	if err != nil {
		return time.Time{}, fmt.Errorf("error querying last called time: %w", err)
	}

	// Look for the last timestamp in the returned series.
	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		series := resp.Results[0].Series[0]
		if len(series.Values) > 0 {
			// The first column is "time" (epoch in ms)
			// Use the last value in the list.
			lastVal := series.Values[len(series.Values)-1][0]
			if ts, ok := lastVal.(float64); ok {
				return time.Unix(0, int64(ts)*int64(time.Millisecond)), nil
			}
		}
	}

	return time.Time{}, nil
}

// GetRequestCount fetches the total number of requests for the given app
// over the specified time window.
// The timeWindow parameter is a string (e.g. "24h", "3d") to define the lookback period.
func (c *Client) GetRequestCount(ctx context.Context, orgID, envID string, app App, timeWindow string) (int, error) {
	templateCH1 := `SELECT sum("avg_request_count") FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`
	templateRTF := `SELECT sum("avg_request_count") FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "cluster_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`
	var query string

	if FilterCH1(app) {
		query = fmt.Sprintf(
			templateCH1,
			orgID, envID, app.Details.Domain, timeWindow,
		)
	} else if FilterRTF(app) {
		query = fmt.Sprintf(
			templateRTF,
			orgID, envID, app.Target.ID, app.Artifact.Name, timeWindow,
		)
	} else {
		return 0, fmt.Errorf("unsupported app type: %s", app.Target.Type)
	}

	params := QueryParams{
		OrgID:      orgID,
		EnvID:      envID,
		AppID:      app.ID,
		Query:      query,
		InfluxDBId: c.InfluxDbId,
	}

	resp, err := c.queryInfluxDB(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error querying request count: %w", err)
	}

	total := 0
	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		series := resp.Results[0].Series[0]
		for _, entry := range series.Values {
			if countVal, ok := entry[1].(float64); ok {
				total += int(countVal)
			}
		}
		return total, nil
	}

	return 0, nil
}
