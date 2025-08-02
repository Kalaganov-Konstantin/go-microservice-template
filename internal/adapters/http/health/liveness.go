package health

import (
	"net/http"
	"time"

	"microservice/internal/adapters/http/response"
)

type LivenessHandler struct {
	version string
}

func NewLivenessHandler(version string) *LivenessHandler {
	return &LivenessHandler{
		version: version,
	}
}

func (h *LivenessHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	select {
	case <-ctx.Done():
		response.RespondError(w, http.StatusRequestTimeout, ctx.Err())
		return
	default:
		livenessResponse := LivenessResponse{
			Status:    StatusPass,
			Timestamp: time.Now(),
			Version:   h.version,
		}
		response.RespondJSON(w, http.StatusOK, livenessResponse)
	}
}
