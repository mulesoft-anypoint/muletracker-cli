package anypoint

import (
	"context"
	"errors"
	"io"

	"github.com/mulesoft-anypoint/anypoint-client-go/exchange_apps"
)

// Get Exchange Client Apps
func (c *Client) GetExchangeClientApps(ctx context.Context, orgID string, targetAdminSite bool) ([]exchange_apps.GetExchangeAppsResponseInner, error) {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_apps.ContextAccessToken, c.getEffectiveToken()), exchange_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_apps.NewAPIClient(exchange_apps.NewConfiguration())
	limit := 250
	page := 0
	result := make([]exchange_apps.GetExchangeAppsResponseInner, 0)
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
func (c *Client) GetExchangeClientAppContracts(ctx context.Context, orgID string, appID int32) ([]exchange_apps.GetExchangeAppContractsResponseInner, error) {
	exchAppCtx := context.WithValue(context.WithValue(ctx, exchange_apps.ContextAccessToken, c.getEffectiveToken()), exchange_apps.ContextServerIndex, c.ServerIndex)
	exchAppClient := exchange_apps.NewAPIClient(exchange_apps.NewConfiguration())
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
