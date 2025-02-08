package anypoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/authorization"
	"github.com/mulesoft-anypoint/anypoint-client-go/org"
	"github.com/mulesoft-anypoint/muletracker-cli/config"
	"github.com/spf13/viper"
)

// Servers mapping for building the base URL dynamically.
var anypointServers = []string{
	"https://anypoint.mulesoft.com",
	"https://eu1.anypoint.mulesoft.com",
	"https://gov.anypoint.mulesoft.com",
}

// Client wraps the anypoint-client-go Client with additional context as needed.
type Client struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
	ServerIndex  int
	ExpiresAt    time.Time // the time when the access token expires
	InfluxDbId   int       // the InfluxDB ID for the organization
}

// NewClient authenticates and returns a new Client instance.
func NewClient(ctx context.Context, serverIndex int, clientId, clientSecret string) (*Client, error) {
	// This is pseudo-code; refer to anypoint-client-go documentation for actual usage.
	creds := authorization.NewCredentialsWithDefaults()
	creds.SetClientId(clientId)
	creds.SetClientSecret(clientSecret)
	apiClient := authorization.NewAPIClient(authorization.NewConfiguration())
	res, httpr, err := apiClient.DefaultApi.ApiV2Oauth2TokenPost(ctx).Credentials(*creds).Execute()
	if err != nil {
		var details string
		if httpr != nil {
			b, _ := io.ReadAll(httpr.Body)
			details = string(b)
		} else {
			details = err.Error()
		}
		return nil, errors.New("error authenticating: " + details)
	}
	defer httpr.Body.Close()

	// Calculate the token expiration time.
	expiresIn := res.GetExpiresIn() // expiresIn is in seconds.
	expirationTime := time.Now().Add(time.Duration(expiresIn) * time.Second)
	client := &Client{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		AccessToken:  res.GetAccessToken(),
		ServerIndex:  serverIndex,
		ExpiresAt:    expirationTime,
	}
	// Retrieve the InfluxDB ID from bootdata.
	_, err = client.GetInfluxDBID(ctx)
	if err != nil {
		return nil, errors.New("error retrieving InfluxDB ID: " + err.Error())
	}
	// Optionally, you might log or print the expiration for debugging:
	// fmt.Printf("Access token will expire at: %s\n", expirationTime.Format(time.RFC1123))
	// You store the client in a global context for later retrieval.
	setGlobalClient(client)
	return client, nil
}

// For simplicity, we store the client globally.
// In a production app, you’d likely use proper dependency injection or context management.
var globalClient *Client

func setGlobalClient(client *Client) {
	// Persist configuration values using Viper.
	// In production, consider more secure storage for sensitive values.
	viper.Set("clientId", client.ClientId)
	viper.Set("clientSecret", client.ClientSecret)
	viper.Set("serverIndex", client.ServerIndex)
	viper.Set("accessToken", client.AccessToken)
	// Persist the expiration time in RFC3339 format.
	viper.Set("expiresAt", client.ExpiresAt.Format(time.RFC3339))
	viper.Set("influxdbId", client.InfluxDbId)
	if err := config.SaveConfig(); err != nil {
		fmt.Printf("Warning: Unable to persist configuration: %v\n", err)
	}
	globalClient = client
}

// GetClientFromContext retrieves the global client.
// If the global client is nil, it attempts to read persisted configuration from Viper
// and recreate the client if the stored token is still valid.
func GetClientFromContext() (*Client, error) {
	// If the global client is already initialized, return it.
	if globalClient != nil {
		return globalClient, nil
	}

	// Attempt to read persisted configuration using Viper.
	clientId := viper.GetString("clientId")
	clientSecret := viper.GetString("clientSecret")
	serverIndex := viper.GetInt("serverIndex")
	accessToken := viper.GetString("accessToken")
	expiresAtStr := viper.GetString("expiresAt")
	influxDbId := viper.GetInt("influxdbId")

	// Check that all required configuration values are available.
	if clientId == "" || clientSecret == "" || accessToken == "" || expiresAtStr == "" || influxDbId == 0 {
		return nil, errors.New("client configuration incomplete. Please run 'connect' command first")
	}

	// Parse the expiration time.
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid expiration time in configuration: %w", err)
	}

	// Check if the token is still valid.
	if time.Now().After(expiresAt) {
		return nil, errors.New("access token expired. Please run 'connect' command")
	}

	// Recreate and store the client from configuration.
	globalClient = &Client{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
		ServerIndex:  serverIndex,
		ExpiresAt:    expiresAt,
		InfluxDbId:   influxDbId,
	}
	return globalClient, nil
}

// get the base URL for the Anypoint Platform API.
func (c *Client) getServerHost() (string, error) {
	if c.ServerIndex < 0 || c.ServerIndex >= len(anypointServers) {
		return "", errors.New("invalid server index")
	}
	return anypointServers[c.ServerIndex], nil
}

// GetBusinessGroups retrieves the business groups.
func (c *Client) GetBusinessGroup(ctx context.Context, orgId string) (*org.MasterBGDetail, error) {
	orgCtx := context.WithValue(context.WithValue(ctx, org.ContextAccessToken, c.AccessToken), org.ContextServerIndex, c.ServerIndex)
	orgClient := org.NewAPIClient(org.NewConfiguration())
	org, httpr, err := orgClient.DefaultApi.OrganizationsOrgIdGet(orgCtx, orgId).Execute()
	if err != nil {
		var details string
		if httpr != nil && httpr.StatusCode >= 400 {
			defer httpr.Body.Close()
			b, _ := io.ReadAll(httpr.Body)
			details = string(b)
		} else {
			details = err.Error()
		}
		return nil, errors.New("error retrieving business groups: " + details)
	}
	defer httpr.Body.Close()
	return &org, nil
}

// GetEnvironments retrieves environments for a given business group ID.
func (c *Client) GetEnvironments(ctx context.Context, bgId string) ([]org.Environment, error) {
	org, err := c.GetBusinessGroup(ctx, bgId)
	if err != nil {
		return nil, err
	}
	return org.GetEnvironments(), nil
}

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
