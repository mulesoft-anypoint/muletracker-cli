package anypoint

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/mulesoft-anypoint/anypoint-client-go/apim"
	"github.com/mulesoft-anypoint/anypoint-client-go/apim_contract"
	"github.com/mulesoft-anypoint/anypoint-client-go/apim_policy"
)

func (c *Client) GetApi(ctx context.Context, orgID, envID, apiId string) (*apim.ApimInstanceDetails, error) {
	apimCtx := context.WithValue(context.WithValue(ctx, apim.ContextAccessToken, c.getEffectiveToken()), apim.ContextServerIndex, c.ServerIndex)
	apimClient := apim.NewAPIClient(apim.NewConfiguration())
	api, httpr, err := apimClient.DefaultApi.GetApimInstanceDetails(apimCtx, orgID, envID, apiId).Execute()
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

	return api, nil
}

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

func (c *Client) GetApiContracts(ctx context.Context, orgID, envID, apiID string) ([]apim_contract.Contract, error) {
	apimContractCtx := context.WithValue(context.WithValue(ctx, apim_contract.ContextAccessToken, c.getEffectiveToken()), apim_contract.ContextServerIndex, c.ServerIndex)
	apimClient := apim_contract.NewAPIClient(apim_contract.NewConfiguration())
	limit := 200
	page := 0
	result := make([]apim_contract.Contract, 0)
	stop := false
	for ok := true; ok; ok = stop {
		contracts, httpr, err := apimClient.DefaultApi.GetApiContracts(apimContractCtx, orgID, envID, apiID).Limit(int32(limit)).Offset(int32(page) * int32(limit)).Execute()
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
		result = append(result, contracts.GetContracts()...)
		stop = len(contracts.GetContracts()) >= limit
		page++
	}
	return result, nil
}

func (c *Client) RevokeApiContract(ctx context.Context, orgID, envID, apiID, contractID string) (*apim_contract.ContractDetails, error) {
	return c.PatchApiContractStatus(ctx, orgID, envID, apiID, contractID, "REVOKE")
}

func (c *Client) ApproveApiContract(ctx context.Context, orgID, envID, apiID, contractID string) (*apim_contract.ContractDetails, error) {
	return c.PatchApiContractStatus(ctx, orgID, envID, apiID, contractID, "APPROVE")
}

func (c *Client) RejectApiContract(ctx context.Context, orgID, envID, apiID, contractID string) (*apim_contract.ContractDetails, error) {
	return c.PatchApiContractStatus(ctx, orgID, envID, apiID, contractID, "REJECT")
}

func (c *Client) RestoreApiContract(ctx context.Context, orgID, envID, apiID, contractID string) (*apim_contract.ContractDetails, error) {
	return c.PatchApiContractStatus(ctx, orgID, envID, apiID, contractID, "RESTORE")
}

func (c *Client) PatchApiContractStatus(ctx context.Context, orgID, envID, apiID, contractID, status string) (*apim_contract.ContractDetails, error) {
	apimContractCtx := context.WithValue(context.WithValue(ctx, apim_contract.ContextAccessToken, c.getEffectiveToken()), apim_contract.ContextServerIndex, c.ServerIndex)
	apimClient := apim_contract.NewAPIClient(apim_contract.NewConfiguration())
	new, httpr, err := apimClient.DefaultApi.UpdateApiContractStatus(apimContractCtx, orgID, envID, apiID, contractID, strings.ToLower(status)).Execute()
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

	return new, nil
}

func (c *Client) DeleteApiContract(ctx context.Context, orgID, envID, apiID, contractID string) error {
	apimContractCtx := context.WithValue(context.WithValue(ctx, apim_contract.ContextAccessToken, c.getEffectiveToken()), apim_contract.ContextServerIndex, c.ServerIndex)
	apimClient := apim_contract.NewAPIClient(apim_contract.NewConfiguration())
	httpr, err := apimClient.DefaultApi.DeleteApiContract(apimContractCtx, orgID, envID, apiID, contractID).Execute()
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

func (c *Client) GetContractDetails(ctx context.Context, orgID, envID, apiID, contractID string) (*apim_contract.ContractDetails, error) {
	apimContractCtx := context.WithValue(context.WithValue(ctx, apim_contract.ContextAccessToken, c.getEffectiveToken()), apim_contract.ContextServerIndex, c.ServerIndex)
	apimClient := apim_contract.NewAPIClient(apim_contract.NewConfiguration())
	res, httpr, err := apimClient.DefaultApi.GetApiContract(apimContractCtx, orgID, envID, apiID, contractID).Execute()
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
	return res, nil
}

func (c *Client) GetApiAsset(ctx context.Context, orgID, envID, apiID string) (*apim.ApimAsset, error) {
	apimAssetCtx := context.WithValue(context.WithValue(ctx, apim.ContextAccessToken, c.getEffectiveToken()), apim.ContextServerIndex, c.ServerIndex)
	apimClient := apim.NewAPIClient(apim.NewConfiguration())
	res, httpr, err := apimClient.DefaultApi.GetApiAsset(apimAssetCtx, orgID, envID, apiID).Execute()
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
	return res, nil
}
