package health

import (
	"context"
	"fmt"
	"microservice/internal/platform/health"
	"net/http"
	"time"
)

type APIChecker struct {
	client   *http.Client
	endpoint string
	name     string
}

func NewAPIChecker(endpoint, name string) *APIChecker {
	return &APIChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		endpoint: endpoint,
		name:     name,
	}
}

func (c *APIChecker) Name() string {
	return c.name
}

func (c *APIChecker) Check(ctx context.Context) health.CheckResult {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint, nil)
	if err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: "failed to create request",
			Error:   err.Error(),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: "api request failed",
			Error:   err.Error(),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return health.CheckResult{
			Status:  health.StatusHealthy,
			Message: fmt.Sprintf("api responding with status %d", resp.StatusCode),
		}
	}

	return health.CheckResult{
		Status:  health.StatusUnhealthy,
		Message: fmt.Sprintf("api returned status %d", resp.StatusCode),
	}
}
