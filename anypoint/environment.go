package anypoint

import (
	"context"
	"errors"
	"io"

	"github.com/mulesoft-anypoint/anypoint-client-go/org"
)

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
