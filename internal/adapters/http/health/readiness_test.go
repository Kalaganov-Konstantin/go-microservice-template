package health

import (
	"encoding/json"
	"microservice/internal/platform/health"
	"microservice/internal/platform/health/mocks"
	"microservice/internal/platform/logger"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewReadinessHandler(t *testing.T) {
	version := "v1.0.0"
	mockManager := mocks.NewMockManagerInterface(t)

	handler := NewReadinessHandler(version, mockManager)

	assert.NotNil(t, handler)
	assert.Equal(t, version, handler.version)
	assert.Equal(t, mockManager, handler.healthManager)
}

func TestReadinessHandler_Check_AllHealthy(t *testing.T) {
	version := "v1.2.3"
	mockManager := mocks.NewMockManagerInterface(t)
	checkResults := map[string]health.CheckResult{
		"database": {
			Status:  health.StatusHealthy,
			Message: "Database connection OK",
		},
		"cache": {
			Status:  health.StatusHealthy,
			Message: "Cache connection OK",
		},
	}
	mockManager.EXPECT().CheckAll(mock.Anything).Return(checkResults).Once()

	handler := NewReadinessHandler(version, mockManager)
	req := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	handler.Check(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, response.Status)
	assert.Equal(t, version, response.Version)
	assert.Len(t, response.Checks, 2)
	assert.Empty(t, response.Notes)

	dbCheck := response.Checks["database"][0]
	assert.Equal(t, "database", dbCheck.ComponentId)
	assert.Equal(t, "dependency", dbCheck.ComponentType)
	assert.Equal(t, StatusPass, dbCheck.Status)
	assert.Equal(t, "Database connection OK", dbCheck.Output)
}

func TestReadinessHandler_Check_WithUnhealthyDependency(t *testing.T) {
	mockManager := mocks.NewMockManagerInterface(t)
	checkResults := map[string]health.CheckResult{
		"database": {
			Status:  health.StatusHealthy,
			Message: "Database connection OK",
		},
		"cache": {
			Status: health.StatusUnhealthy,
			Error:  "Connection timeout",
		},
	}
	mockManager.EXPECT().CheckAll(mock.Anything).Return(checkResults).Once()

	handler := NewReadinessHandler("v1.0.0", mockManager)
	req := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	handler.Check(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, response.Status)
	assert.Len(t, response.Checks, 2)
	assert.Contains(t, response.Notes, "Dependency cache is unavailable")

	cacheCheck := response.Checks["cache"][0]
	assert.Equal(t, StatusFail, cacheCheck.Status)
	assert.Equal(t, "Connection timeout", cacheCheck.Output)
}

func TestReadinessHandler_Check_WithWarningDependency(t *testing.T) {
	mockManager := mocks.NewMockManagerInterface(t)
	checkResults := map[string]health.CheckResult{
		"database": {
			Status:  health.StatusHealthy,
			Message: "Database connection OK",
		},
		"external_api": {
			Status:  "unknown",
			Message: "High latency detected",
		},
	}
	mockManager.EXPECT().CheckAll(mock.Anything).Return(checkResults).Once()

	handler := NewReadinessHandler("v1.0.0", mockManager)
	req := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	handler.Check(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusWarn, response.Status)
	assert.Len(t, response.Checks, 2)

	apiCheck := response.Checks["external_api"][0]
	assert.Equal(t, StatusWarn, apiCheck.Status)
	assert.Equal(t, "High latency detected", apiCheck.Output)
}

func TestReadinessHandler_Check_NoHealthChecks(t *testing.T) {
	mockManager := mocks.NewMockManagerInterface(t)
	checkResults := map[string]health.CheckResult{}
	mockManager.EXPECT().CheckAll(mock.Anything).Return(checkResults).Once()

	handler := NewReadinessHandler("v1.0.0", mockManager)
	req := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	handler.Check(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, response.Status)
	assert.Empty(t, response.Checks)
	assert.Empty(t, response.Notes)
}

func TestReadinessHandler_Check_MixedStatuses(t *testing.T) {
	mockManager := mocks.NewMockManagerInterface(t)
	checkResults := map[string]health.CheckResult{
		"database": {
			Status:  health.StatusHealthy,
			Message: "OK",
		},
		"cache": {
			Status: health.StatusUnhealthy,
			Error:  "Connection failed",
		},
		"metrics": {
			Status:  "unknown",
			Message: "Slow response",
		},
	}
	mockManager.EXPECT().CheckAll(mock.Anything).Return(checkResults).Once()

	handler := NewReadinessHandler("v1.0.0", mockManager)
	req := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	handler.Check(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusFail, response.Status)
	assert.Len(t, response.Checks, 3)
}

func TestCheckDetail_JSONSerialization(t *testing.T) {
	detail := CheckDetail{
		ComponentId:   "test-component",
		ComponentType: "dependency",
		Status:        StatusPass,
		Time:          time.Now(),
		Output:        "All systems operational",
	}

	jsonData, err := json.Marshal(detail)
	require.NoError(t, err)

	var unmarshaled CheckDetail
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, detail.ComponentId, unmarshaled.ComponentId)
	assert.Equal(t, detail.ComponentType, unmarshaled.ComponentType)
	assert.Equal(t, detail.Status, unmarshaled.Status)
	assert.Equal(t, detail.Output, unmarshaled.Output)
	assert.WithinDuration(t, detail.Time, unmarshaled.Time, time.Millisecond)
}

func TestStatus_Constants(t *testing.T) {
	assert.Equal(t, Status("pass"), StatusPass)
	assert.Equal(t, Status("fail"), StatusFail)
	assert.Equal(t, Status("warn"), StatusWarn)
}
