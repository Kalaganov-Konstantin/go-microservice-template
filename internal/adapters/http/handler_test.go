package http

import (
	"errors"
	httpErrors "microservice/internal/platform/http"
	"microservice/internal/platform/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler_Success(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("success"))
		if err != nil {
			return err
		}
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestErrorHandler_HTTPError(t *testing.T) {
	expectedErr := httpErrors.NewBadRequest("Test bad request", errors.New("test error"))
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Test bad request"}`, w.Body.String())
}

func TestErrorHandler_NotFoundError(t *testing.T) {
	expectedErr := httpErrors.NewNotFound("Resource not found", errors.New("not found"))
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error":"Resource not found"}`, w.Body.String())
}

func TestErrorHandler_ConflictError(t *testing.T) {
	expectedErr := httpErrors.NewConflict("Resource conflict", errors.New("conflict"))
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.JSONEq(t, `{"error":"Resource conflict"}`, w.Body.String())
}

func TestErrorHandler_InternalServerError(t *testing.T) {
	expectedErr := httpErrors.NewInternalServerError("Internal error", errors.New("internal"))
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"Internal error"}`, w.Body.String())
}

func TestErrorHandler_UnknownError(t *testing.T) {
	unknownErr := errors.New("some unknown error")
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return unknownErr
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	ErrorHandler(handlerFunc)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
}

func TestErrorHandler_MultipleRequestTypes(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		remoteAddr     string
		handlerFunc    HandlerFunc
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "GET request with error",
			method:     http.MethodGet,
			path:       "/api/test",
			remoteAddr: "192.168.1.1:12345",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return httpErrors.NewBadRequest("Invalid parameter", errors.New("param error"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid parameter"}`,
		},
		{
			name:       "POST request with error",
			method:     http.MethodPost,
			path:       "/api/create",
			remoteAddr: "10.0.0.1:54321",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return httpErrors.NewConflict("Already exists", errors.New("duplicate"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"Already exists"}`,
		},
		{
			name:       "PUT request with success",
			method:     http.MethodPut,
			path:       "/api/update/123",
			remoteAddr: "127.0.0.1:9999",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"status":"updated"}`))
				if err != nil {
					return err
				}
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"updated"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
			w := httptest.NewRecorder()

			ErrorHandler(tt.handlerFunc)(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus != http.StatusOK {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestErrorHandler_ContextWithoutLogger(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("test error without logger")
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	require.NotPanics(t, func() {
		ErrorHandler(handlerFunc)(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
}
