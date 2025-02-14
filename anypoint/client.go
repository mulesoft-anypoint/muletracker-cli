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
	ClientId         string
	ClientSecret     string
	AccessToken      string
	AdminAccessToken string // token given manually which is supposed to be platform admin
	ServerIndex      int
	ExpiresAt        time.Time // the time when the access token expires
	InfluxDbId       int       // the InfluxDB ID for the organization
	Org              string
	Env              string
	// New fields to track which token is being used.
	ActiveTokenType string // "admin" or "connected"
}

// GetClientOptions holds optional parameters for GetClientFromContext.
type GetClientOptions struct {
	SkipTokenExpiration bool
}

// GetClientOption is a function that modifies GetClientOptions.
type GetClientOption func(*GetClientOptions)

// WithSkipTokenExpiration is an option to skip checking token expiration.
func WithSkipTokenExpiration() GetClientOption {
	return func(opts *GetClientOptions) {
		opts.SkipTokenExpiration = true
	}
}

// NewClient authenticates and returns a new Client instance.
func NewClient(ctx context.Context, serverIndex int, clientId, clientSecret string) (*Client, error) {
	loginRes, err := loginConnectedApp(ctx, serverIndex, clientId, clientSecret)
	if err != nil {
		return nil, errors.New("error authenticating: " + err.Error())
	}
	// Calculate the token expiration time.
	expiresIn := loginRes.GetExpiresIn() // expiresIn is in seconds.
	expirationTime := time.Now().Add(time.Duration(expiresIn) * time.Second)
	client := &Client{
		ClientId:        clientId,
		ClientSecret:    clientSecret,
		AccessToken:     loginRes.GetAccessToken(),
		ServerIndex:     serverIndex,
		ExpiresAt:       expirationTime,
		Org:             viper.GetString("org"),
		Env:             viper.GetString("env"),
		ActiveTokenType: "connected",
	}
	// Retrieve the InfluxDB ID from bootdata.
	_, err = client.GetInfluxDBID(ctx)
	if err != nil {
		return nil, errors.New("error retrieving InfluxDB ID: " + err.Error())
	}
	// You store the client in a global context for later retrieval.
	setGlobalClient(client)
	return client, nil
}

// Logs in the connected app
func loginConnectedApp(ctx context.Context, serverIndex int, clientId, clientSecret string) (*authorization.InlineResponse200, error) {
	authCtx := context.WithValue(ctx, authorization.ContextServerIndex, serverIndex)
	creds := authorization.NewCredentialsWithDefaults()
	creds.SetClientId(clientId)
	creds.SetClientSecret(clientSecret)
	apiClient := authorization.NewAPIClient(authorization.NewConfiguration())
	res, httpr, err := apiClient.DefaultApi.ApiV2Oauth2TokenPost(authCtx).Credentials(*creds).Execute()
	if err != nil {
		var details string
		if httpr != nil {
			b, _ := io.ReadAll(httpr.Body)
			details = string(b)
		} else {
			details = err.Error()
		}
		return nil, errors.New(details)
	}
	defer httpr.Body.Close()
	return &res, nil
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
	viper.Set("adminAccessToken", client.AdminAccessToken)
	viper.Set("expiresAt", client.ExpiresAt.Format(time.RFC3339))
	viper.Set("influxdbId", client.InfluxDbId)
	viper.Set("org", client.Org)
	viper.Set("env", client.Env)
	viper.Set("activeTokenType", client.ActiveTokenType)

	//Save conf
	if err := config.SaveConfig(); err != nil {
		fmt.Printf("Warning: Unable to persist configuration: %v\n", err)
	}
	globalClient = client
}

// GetClientFromContext retrieves the global client.
// If the global client is nil, it attempts to read persisted configuration from Viper
// and recreate the client if the stored token is still valid.
func GetClientFromContext(opts ...GetClientOption) (*Client, error) {
	// Set default options.
	options := &GetClientOptions{
		SkipTokenExpiration: false,
	}

	// Apply all provided options.
	for _, opt := range opts {
		opt(options)
	}

	// If globalClient is available, check token expiration unless it's skipped.
	if globalClient != nil {
		if !options.SkipTokenExpiration && time.Now().After(globalClient.ExpiresAt) {
			return nil, errors.New("token expired; please run 'connect' command")
		}
		return globalClient, nil
	}

	// Attempt to read persisted configuration using Viper.
	clientId := viper.GetString("clientId")
	clientSecret := viper.GetString("clientSecret")
	serverIndex := viper.GetInt("serverIndex")
	accessToken := viper.GetString("accessToken")
	adminAccessToken := viper.GetString("adminAccessToken")
	activeTokenType := viper.GetString("activeTokenType")
	expiresAtStr := viper.GetString("expiresAt")
	influxDbId := viper.GetInt("influxdbId")
	org := viper.GetString("org")
	env := viper.GetString("env")

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
	if (activeTokenType == "connected" || !options.SkipTokenExpiration) && isTokenExpired(expiresAt) {
		return nil, errors.New("access token expired. Please run 'connect' command")
	}

	// Recreate and store the client from configuration.
	globalClient = &Client{
		ClientId:         clientId,
		ClientSecret:     clientSecret,
		AccessToken:      accessToken,
		AdminAccessToken: adminAccessToken,
		ServerIndex:      serverIndex,
		ExpiresAt:        expiresAt,
		InfluxDbId:       influxDbId,
		Org:              org,
		Env:              env,
		ActiveTokenType:  activeTokenType,
	}
	return globalClient, nil
}

func isTokenExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

func (c *Client) SetOrg(org string) {
	c.Org = org
	setGlobalClient(c)
}

func (c *Client) SetEnv(env string) {
	c.Env = env
	setGlobalClient(c)
}

func (c *Client) SetAdminAccessToken(accessToken string) {
	c.ActiveTokenType = "admin"
	c.AdminAccessToken = accessToken
	setGlobalClient(c)
}

func (c *Client) IsOrgEmpty() bool {
	return len(c.Org) == 0
}

func (c *Client) IsEnvEmpty() bool {
	return len(c.Env) == 0
}

// get the base URL for the Anypoint Platform API.
func (c *Client) getServerHost() (string, error) {
	if c.ServerIndex < 0 || c.ServerIndex >= len(anypointServers) {
		return "", errors.New("invalid server index")
	}
	return anypointServers[c.ServerIndex], nil
}

// returns the token to use
func (c *Client) getEffectiveToken() string {
	if c.ActiveTokenType != "admin" {
		return c.AdminAccessToken
	}
	return c.AccessToken
}

// GetBusinessGroups retrieves the business groups.
func (c *Client) GetBusinessGroup(ctx context.Context, orgId string) (*org.MasterBGDetail, error) {
	orgCtx := context.WithValue(context.WithValue(ctx, org.ContextAccessToken, c.getEffectiveToken()), org.ContextServerIndex, c.ServerIndex)
	orgClient := org.NewAPIClient(org.NewConfiguration())
	orgResult, httpr, err := orgClient.DefaultApi.OrganizationsOrgIdGet(orgCtx, orgId).Execute()
	if err != nil {
		var details string
		if httpr != nil && httpr.StatusCode >= 400 {
			defer httpr.Body.Close()
			b, _ := io.ReadAll(httpr.Body)
			details = string(b)
		} else {
			details = err.Error()
		}
		return nil, errors.New(details)
	}
	defer httpr.Body.Close()
	return &orgResult, nil
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
