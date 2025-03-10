package anypoint

import (
	"context"
	"errors"
	"io"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim"
	"github.com/mulesoft-anypoint/anypoint-client-go/apim_policy"
)

func (c *Client) GetApis(ctx context.Context, orgID, envID string, limit, offset int) ([]apim.ApimInstanceCollectionAssetsInner, error) {
	apimCtx := context.WithValue(context.WithValue(ctx, apim.ContextAccessToken, c.getEffectiveToken()), apim.ContextServerIndex, c.ServerIndex)
	apimClient := apim.NewAPIClient(apim.NewConfiguration())
	apis, httpr, err := apimClient.DefaultApi.GetEnvApimInstances(apimCtx, orgID, envID).Limit(int32(limit)).Offset(int32(offset)).Execute()
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

	return apis.GetAssets(), nil
}

func (c *Client) GetAllApis(ctx context.Context, orgID, envID string) ([]apim.ApimInstanceCollectionAssetsInner, error) {
	limit := 50
	page := 1
	stop := false
	result := make([]apim.ApimInstanceCollectionAssetsInner, 0)
	for ok := true; ok; ok = (page > 0 && !stop) {
		list, err := c.GetApis(ctx, orgID, envID, limit, (page-1)*limit)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
		page++
		stop = len(list) < limit
	}
	return result, nil
}

func (c *Client) DeleteApi(ctx context.Context, orgID, envID, apiID string) error {
	apimCtx := context.WithValue(context.WithValue(ctx, apim.ContextAccessToken, c.getEffectiveToken()), apim.ContextServerIndex, c.ServerIndex)
	apimClient := apim.NewAPIClient(apim.NewConfiguration())
	httpr, err := apimClient.DefaultApi.DeleteApimInstance(apimCtx, orgID, envID, apiID).Execute()
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

func (c *Client) GetApiPolicies(ctx context.Context, orgID, envID, apiID string) ([]apim_policy.ApimPolicy, error) {
	apimPolicyCtx := context.WithValue(context.WithValue(ctx, apim_policy.ContextAccessToken, c.getEffectiveToken()), apim_policy.ContextServerIndex, c.ServerIndex)
	apimClient := apim_policy.NewAPIClient(apim_policy.NewConfiguration())
	res, httpr, err := apimClient.DefaultApi.GetApimPolicies(apimPolicyCtx, orgID, envID, apiID).Execute()
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

	return *res.ArrayOfApimPolicy, nil
}
