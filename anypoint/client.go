package anypoint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mulesoft-anypoint/anypoint-client-go/authorization"
	"github.com/mulesoft-anypoint/anypoint-client-go/org"
)

// Client wraps the anypoint-client-go Client with additional context as needed.
type Client struct {
	AccessToken string
	ServerIndex int
	ExpiresAt   time.Time // the time when the access token expires
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
		AccessToken: res.GetAccessToken(),
		ServerIndex: serverIndex,
		ExpiresAt:   expirationTime,
	}
	// Optionally, you might log or print the expiration for debugging:
	fmt.Printf("Access token will expire at: %s\n", expirationTime.Format(time.RFC1123))
	// You might store the client in a global context or similar for later retrieval.
	setGlobalClient(client)
	return client, nil
}

// For simplicity, we store the client globally.
// In a production app, you’d likely use proper dependency injection or context management.
var globalClient *Client

func setGlobalClient(client *Client) {
	globalClient = client
}

// GetClientFromContext retrieves the global client.
func GetClientFromContext() (*Client, error) {
	if globalClient == nil {
		return nil, errors.New("client not initialized. Please run 'muletracker connect' first")
	}
	return globalClient, nil
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

// GetLastCalledTime fetches the last time the given app was called.
// It uses a query that calculates the 75th percentile of the avg_response_time
// over the specified time window. It returns the timestamp of the latest data point.
// The timeWindow parameter is a string (e.g. "15m", "24h", "3d") to define the lookback period.
func (c *Client) GetLastCalledTime(ctx context.Context, orgID, envID, appID, timeWindow string) (time.Time, error) {
	query := fmt.Sprintf(
		`SELECT percentile("avg_response_time", 75) FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`,
		orgID, envID, appID, timeWindow,
	)

	params := QueryParams{
		OrgID: orgID,
		EnvID: envID,
		AppID: appID,
		Query: query,
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

	return time.Time{}, errors.New("no data found for last called time")
}

// GetRequestCount fetches the total number of requests for the given app
// over the specified time window.
// The timeWindow parameter is a string (e.g. "24h", "3d") to define the lookback period.
func (c *Client) GetRequestCount(ctx context.Context, orgID, envID, appID, timeWindow string) (int, error) {
	query := fmt.Sprintf(
		`SELECT count("avg_response_time") FROM "app_inbound_metric" WHERE "org_id" = '%s' AND "env_id" = '%s' AND "app_id" = '%s' AND time >= now() - %s GROUP BY time(1m), "app_id" fill(none) tz('Europe/Paris')`,
		orgID, envID, appID, timeWindow,
	)

	params := QueryParams{
		OrgID: orgID,
		EnvID: envID,
		AppID: appID,
		Query: query,
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

	return 0, errors.New("no data found for request count")
}
