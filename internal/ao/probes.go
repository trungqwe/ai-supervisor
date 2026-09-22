package ao

import (
	"context"
	"fmt"
	"strings"
)

// CheckHealth queries GET /healthz to verify AO daemon liveness.
// Success requires HTTP 200 and status == "ok".
func (c *Client) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	var resp HealthStatus
	if err := c.get(ctx, "/healthz", &resp); err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, &ProtocolError{
			StatusCode: 200,
			Method:     "GET",
			Path:       "/healthz",
			Reason:     fmt.Sprintf("health probe status %q is not ok", resp.Status),
		}
	}
	return &resp, nil
}

// CheckReadiness queries GET /readyz to verify AO daemon readiness.
// Success requires HTTP 200 and status == "ready".
func (c *Client) CheckReadiness(ctx context.Context) (*ReadinessStatus, error) {
	var resp ReadinessStatus
	if err := c.get(ctx, "/readyz", &resp); err != nil {
		return nil, err
	}
	if resp.Status != "ready" {
		return nil, &ProtocolError{
			StatusCode: 200,
			Method:     "GET",
			Path:       "/readyz",
			Reason:     fmt.Sprintf("readiness probe status %q is not ready", resp.Status),
		}
	}
	return &resp, nil
}

// ListAgents queries GET /api/v1/agents to inspect the available agent harness catalog.
func (c *Client) ListAgents(ctx context.Context) (*AgentInventory, error) {
	var resp AgentInventory
	if err := c.get(ctx, "/api/v1/agents", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentReadiness queries GET /api/v1/agents/readiness and extracts the readiness snapshot
// for the specified agentID. Empty agentID returns ErrBadRequest. If agentID is not found, ErrAgentNotFound is returned.
func (c *Client) GetAgentReadiness(ctx context.Context, agentID string) (*AgentReadinessSnapshot, error) {
	trimmedID := strings.TrimSpace(agentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: agentID cannot be empty", ErrBadRequest)
	}

	var resp rawAgentReadinessResponse
	if err := c.get(ctx, "/api/v1/agents/readiness", &resp); err != nil {
		return nil, err
	}

	for _, agent := range resp.Agents {
		if agent.ID == trimmedID {
			return &agent, nil
		}
	}

	return nil, fmt.Errorf("%w: agent %q", ErrAgentNotFound, trimmedID)
}

// GetAPIContract queries GET /api/v1/openapi.yaml and returns the raw schema content.
// Note: per ADR-011, this schema is an API compatibility signal only and does not prove the binary release version.
func (c *Client) GetAPIContract(ctx context.Context) (string, error) {
	schema, err := c.getRaw(ctx, "/api/v1/openapi.yaml", "application/yaml, text/yaml, text/plain")
	if err != nil {
		return "", err
	}
	return schema, nil
}
