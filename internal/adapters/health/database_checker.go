package health

import (
	"context"
	"microservice/internal/platform/health"

	"microservice/internal/adapters/database"
)

type DatabaseChecker struct {
	db   *database.Lifecycle
	name string
}

func NewDatabaseChecker(db *database.Lifecycle, name string) *DatabaseChecker {
	return &DatabaseChecker{
		db:   db,
		name: name,
	}
}

func (c *DatabaseChecker) Name() string {
	return c.name
}

func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	db := c.db.Connection()
	if db == nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: "database connection is not initialized",
		}
	}

	err := db.Ping(ctx)
	if err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: "database connection failed",
			Error:   err.Error(),
		}
	}

	return health.CheckResult{
		Status:  health.StatusHealthy,
		Message: "database connection healthy",
	}
}
