package ao

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// CheckHealth queries GET /healthz to verify AO daemon liveness.
// Success requires HTTP 200, status == "ok", service == "agent-orchestrator-daemon", and pid > 0.
func (c *Client) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	var wire wireDaemonProbe
	if err := c.get(ctx, "/healthz", http.StatusOK, &wire); err != nil {
		return nil, err
	}
	if wire.Status != "ok" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/healthz",
			Reason:     fmt.Sprintf("health probe status %q is not ok", wire.Status),
		}
	}
	if wire.Service != DaemonServiceAO {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/healthz",
			Reason:     fmt.Sprintf("health probe service %q does not match expected %q", wire.Service, DaemonServiceAO),
		}
	}
	if wire.PID <= 0 {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/healthz",
			Reason:     fmt.Sprintf("health probe pid %d is not a positive process ID", wire.PID),
		}
	}
	return toNormalizedHealthStatus(&wire), nil
}

// CheckReadiness queries GET /readyz to verify AO daemon readiness.
// Success requires HTTP 200, status == "ready", service == "agent-orchestrator-daemon", and pid > 0.
func (c *Client) CheckReadiness(ctx context.Context) (*ReadinessStatus, error) {
	var wire wireDaemonProbe
	if err := c.get(ctx, "/readyz", http.StatusOK, &wire); err != nil {
		return nil, err
	}
	if wire.Status != "ready" {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/readyz",
			Reason:     fmt.Sprintf("readiness probe status %q is not ready", wire.Status),
		}
	}
	if wire.Service != DaemonServiceAO {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/readyz",
			Reason:     fmt.Sprintf("readiness probe service %q does not match expected %q", wire.Service, DaemonServiceAO),
		}
	}
	if wire.PID <= 0 {
		return nil, &ProtocolError{
			StatusCode: http.StatusOK,
			Method:     http.MethodGet,
			Path:       "/readyz",
			Reason:     fmt.Sprintf("readiness probe pid %d is not a positive process ID", wire.PID),
		}
	}
	return toNormalizedReadinessStatus(&wire), nil
}

// ListAgents queries GET /api/v1/agents to inspect the available agent harness catalog.
// Success requires HTTP 200.
func (c *Client) ListAgents(ctx context.Context) (*AgentInventory, error) {
	var wire wireAgentInventory
	if err := c.get(ctx, "/api/v1/agents", http.StatusOK, &wire); err != nil {
		return nil, err
	}
	return toNormalizedAgentInventory(&wire), nil
}

// GetAgentReadiness queries GET /api/v1/agents/readiness and extracts the readiness snapshot
// for the specified agentID. Empty agentID returns ErrBadRequest. If agentID is not found, ErrAgentNotFound is returned.
// Success requires HTTP 200.
func (c *Client) GetAgentReadiness(ctx context.Context, agentID string) (*AgentReadinessSnapshot, error) {
	if strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("%w: agentID cannot be empty or whitespace", ErrBadRequest)
	}

	var wire wireAgentReadinessResponse
	if err := c.get(ctx, "/api/v1/agents/readiness", http.StatusOK, &wire); err != nil {
		return nil, err
	}

	for _, agent := range wire.Agents {
		if agent.ID == agentID {
			return toNormalizedAgentReadiness(&agent), nil
		}
	}

	return nil, fmt.Errorf("%w: agent %q", ErrAgentNotFound, agentID)
}

// GetAPIContract queries GET /api/v1/openapi.yaml and returns the raw schema content.
// Success requires HTTP 200.
// Note: per ADR-011, this schema is an API compatibility signal only and does not prove the binary release version.
func (c *Client) GetAPIContract(ctx context.Context) (string, error) {
	schema, err := c.getRaw(ctx, "/api/v1/openapi.yaml", http.StatusOK, "application/yaml, text/yaml, text/plain")
	if err != nil {
		return "", err
	}
	return schema, nil
}
