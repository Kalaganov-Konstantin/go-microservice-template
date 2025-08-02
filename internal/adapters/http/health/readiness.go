package health

import (
	"context"
	"microservice/internal/platform/health"
	"microservice/internal/platform/logger"
	"net/http"
	"time"

	"microservice/internal/adapters/http/response"
)

type ReadinessHandler struct {
	version       string
	healthManager health.ManagerInterface
}

func NewReadinessHandler(version string, healthManager health.ManagerInterface) *ReadinessHandler {
	return &ReadinessHandler{
		version:       version,
		healthManager: healthManager,
	}
}

func (h *ReadinessHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	log := logger.FromContext(ctx)
	healthResults := h.healthManager.CheckAll(ctx)
	overallStatus := StatusPass
	checks := make(map[string][]CheckDetail)
	var notes []string

	for name, result := range healthResults {
		var status Status
		switch result.Status {
		case health.StatusHealthy:
			status = StatusPass
		case health.StatusUnhealthy:
			status = StatusFail
			overallStatus = StatusFail
		default:
			status = StatusWarn
			if overallStatus == StatusPass {
				overallStatus = StatusWarn
			}
		}

		checkDetail := CheckDetail{
			ComponentId:   name,
			ComponentType: "dependency",
			Status:        status,
			Time:          time.Now(),
			Output:        result.Message,
		}

		if result.Error != "" {
			checkDetail.Output = result.Error
		}

		checks[name] = []CheckDetail{checkDetail}

		if status == StatusFail {
			notes = append(notes, "Dependency "+name+" is unavailable")
		}
	}

	readinessResponse := ReadinessResponse{
		Status:  overallStatus,
		Version: h.version,
		Checks:  checks,
		Notes:   notes,
	}

	statusCode := http.StatusOK
	if overallStatus == StatusFail {
		statusCode = http.StatusServiceUnavailable
		log.Warn("Readiness check failed", logger.String("status", string(overallStatus)))
	}

	response.RespondJSON(w, statusCode, readinessResponse)
}
