package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLivenessHandler(t *testing.T) {
	version := "v1.0.0"
	handler := NewLivenessHandler(version)

	assert.NotNil(t, handler)
	assert.Equal(t, version, handler.version)
}

func TestLivenessHandler_Check(t *testing.T) {
	version := "v1.2.3"
	handler := NewLivenessHandler(version)
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	w := httptest.NewRecorder()

	testStart := time.Now()

	handler.Check(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response LivenessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, response.Status)
	assert.Equal(t, version, response.Version)
	assert.WithinDuration(t, testStart, response.Timestamp, 2*time.Second)
}

func TestLivenessHandler_Check_WithEmptyVersion(t *testing.T) {
	handler := NewLivenessHandler("")
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	w := httptest.NewRecorder()

	testStart := time.Now()

	handler.Check(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response LivenessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, response.Status)
	assert.Empty(t, response.Version)
	assert.WithinDuration(t, testStart, response.Timestamp, 2*time.Second)
}

func TestLivenessHandler_Check_MultipleRequests(t *testing.T) {
	handler := NewLivenessHandler("v1.0.0")

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
		w := httptest.NewRecorder()

		handler.Check(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LivenessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, StatusPass, response.Status)
	}
}

func TestLivenessResponse_JSONFields(t *testing.T) {
	timestamp := time.Now()
	response := LivenessResponse{
		Status:    StatusPass,
		Timestamp: timestamp,
		Version:   "v1.0.0",
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled LivenessResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, StatusPass, unmarshaled.Status)
	assert.Equal(t, "v1.0.0", unmarshaled.Version)
	assert.WithinDuration(t, timestamp, unmarshaled.Timestamp, time.Millisecond)
}
