package ao

import (
	"context"
	"fmt"
)

// CheckHealth queries GET /healthz to verify AO daemon liveness.
func (c *Client) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	var resp HealthStatus
	if err := c.get(ctx, "/healthz", &resp); err != nil {
		return nil, fmt.Errorf("check health probe failed: %w", err)
	}
	return &resp, nil
}

// CheckReadiness queries GET /readyz to verify AO daemon readiness.
func (c *Client) CheckReadiness(ctx context.Context) (*ReadinessStatus, error) {
	var resp ReadinessStatus
	if err := c.get(ctx, "/readyz", &resp); err != nil {
		return nil, fmt.Errorf("check readiness probe failed: %w", err)
	}
	return &resp, nil
}

// ListAgents queries GET /api/v1/agents to inspect available agent harness catalog.
func (c *Client) ListAgents(ctx context.Context) (*AgentInventory, error) {
	var resp AgentInventory
	if err := c.get(ctx, "/api/v1/agents", &resp); err != nil {
		return nil, fmt.Errorf("list agents failed: %w", err)
	}
	return &resp, nil
}

// GetAgentReadiness queries GET /api/v1/agents/readiness to inspect agent harness readiness.
func (c *Client) GetAgentReadiness(ctx context.Context) (*AgentReadinessResponse, error) {
	var resp AgentReadinessResponse
	if err := c.get(ctx, "/api/v1/agents/readiness", &resp); err != nil {
		return nil, fmt.Errorf("get agent readiness failed: %w", err)
	}
	return &resp, nil
}

// GetAPIContract queries GET /api/v1/openapi.yaml to fetch the OpenAPI contract schema.
// Note: per ADR-011 and P03 reconciliation, this schema serves as an API compatibility signal
// only and does not prove the specific AO daemon release version.
func (c *Client) GetAPIContract(ctx context.Context) (string, error) {
	schema, err := c.getRaw(ctx, "/api/v1/openapi.yaml", "application/yaml, text/yaml, text/plain")
	if err != nil {
		return "", fmt.Errorf("get API contract failed: %w", err)
	}
	return schema, nil
}
