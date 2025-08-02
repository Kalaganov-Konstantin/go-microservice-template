package health

import (
	"context"
	"microservice/internal/platform/health"
)

type MemoryChecker struct{}

func NewMemoryChecker() *MemoryChecker {
	return &MemoryChecker{}
}

func (c *MemoryChecker) Name() string {
	return "memory_storage"
}

func (c *MemoryChecker) Check(ctx context.Context) health.CheckResult {
	select {
	case <-ctx.Done():
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: "memory storage check cancelled",
			Error:   ctx.Err().Error(),
		}
	default:
		return health.CheckResult{
			Status:  health.StatusHealthy,
			Message: "memory storage operational",
		}
	}
}
