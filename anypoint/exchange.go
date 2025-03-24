package anypoint

import (
	"context"
	"errors"
	"io"

	"github.com/mulesoft-anypoint/anypoint-client-go/exchange_client_apps"
)

func (c *Client) PostExchangeClientApp(ctx context.Context, orgID, name, description, url string, grantTypes, redirectUris []string) (*exchange_client_apps.ClientApp, error) {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_client_apps.ContextAccessToken, c.getEffectiveToken()), exchange_client_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_client_apps.NewAPIClient(exchange_client_apps.NewConfiguration())
	body := exchange_client_apps.NewPostExchangeAppsBodyWithDefaults()
	body.SetName(name)
	body.SetDescription(description)
	body.SetGrantTypes(grantTypes)
	body.SetApiEndpoints(false)
	body.SetRedirectUri(redirectUris)
	body.SetUrl(url)
	app, httpr, err := exchAppClient.DefaultApi.PostExchangeClientApp(exchAppCtx, orgID).PostExchangeAppsBody(*body).Execute()
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

	return app, nil
}

// Get Exchange Client Apps
func (c *Client) GetExchangeClientApps(ctx context.Context, orgID string, targetAdminSite bool) ([]exchange_client_apps.ClientApp, error) {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_client_apps.ContextAccessToken, c.getEffectiveToken()), exchange_client_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_client_apps.NewAPIClient(exchange_client_apps.NewConfiguration())
	limit := 200
	page := 0
	result := make([]exchange_client_apps.ClientApp, 0)
	stop := false
	for ok := true; ok; ok = stop {
		exchApps, httpr, err := exchAppClient.DefaultApi.GetExchangeClientApps(exchAppCtx, orgID).Limit(int32(limit)).Offset(int32(limit) * int32(page)).TargetAdminSite(targetAdminSite).Execute()
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
		result = append(result, exchApps...)
		stop = len(exchApps) >= limit
		page++
	}

	return result, nil
}

// Get Exchange Client Application Contracts
func (c *Client) GetExchangeClientAppContracts(ctx context.Context, orgID string, appID int32) ([]exchange_client_apps.ClientAppContract, error) {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_client_apps.ContextAccessToken, c.getEffectiveToken()), exchange_client_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_client_apps.NewAPIClient(exchange_client_apps.NewConfiguration())
	contracts, httpr, err := exchAppClient.DefaultApi.GetExchangeClientAppContracts(exchAppCtx, orgID, appID).Execute()
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

	return contracts, nil
}

// Delete Exchange Client Application
func (c *Client) DeleteExchangeClientApp(ctx context.Context, orgID string, appID int32) error {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_client_apps.ContextAccessToken, c.getEffectiveToken()), exchange_client_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_client_apps.NewAPIClient(exchange_client_apps.NewConfiguration())
	httpr, err := exchAppClient.DefaultApi.DeleteExchangeClientApp(exchAppCtx, orgID, appID).Execute()
	if err != nil {
		var details string
		if httpr != nil && httpr.StatusCode >= 400 {
			defer httpr.Body.Close()
			b, _ := io.ReadAll(httpr.Body)
			details = string(b)
		} else {
			details = err.Error()
		}
		return errors.New(details)
	}
	defer httpr.Body.Close()
	return nil
}
