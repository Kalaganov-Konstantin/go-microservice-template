package http

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomError struct {
	Code int
}

func (c *CustomError) Error() string {
	return fmt.Sprintf("custom error with code %d", c.Code)
}

func TestError_Error_WithMessage(t *testing.T) {
	err := &Error{
		StatusCode: http.StatusBadRequest,
		Message:    "custom error message",
		Err:        errors.New("underlying error"),
	}

	assert.Equal(t, "custom error message", err.Error())
}

func TestError_Error_WithoutMessageButWithErr(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &Error{
		StatusCode: http.StatusBadRequest,
		Message:    "",
		Err:        underlyingErr,
	}

	assert.Equal(t, "underlying error", err.Error())
}

func TestError_Error_WithoutMessageAndErr(t *testing.T) {
	err := &Error{
		StatusCode: http.StatusNotFound,
		Message:    "",
		Err:        nil,
	}

	assert.Equal(t, "Not Found", err.Error())
}

func TestError_Error_WithStatusInternalServerError(t *testing.T) {
	err := &Error{
		StatusCode: http.StatusInternalServerError,
		Message:    "",
		Err:        nil,
	}

	assert.Equal(t, "Internal Server Error", err.Error())
}

func TestError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &Error{
		StatusCode: http.StatusBadRequest,
		Message:    "wrapper message",
		Err:        underlyingErr,
	}

	assert.Equal(t, underlyingErr, err.Unwrap())
}

func TestError_Unwrap_Nil(t *testing.T) {
	err := &Error{
		StatusCode: http.StatusBadRequest,
		Message:    "no underlying error",
		Err:        nil,
	}

	assert.Nil(t, err.Unwrap())
}

func TestNew(t *testing.T) {
	underlyingErr := errors.New("test error")
	err := New(http.StatusBadRequest, "bad request", underlyingErr)

	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "bad request", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
}

func TestNewNotFound(t *testing.T) {
	underlyingErr := errors.New("resource not found")
	err := NewNotFound("Entity not found", underlyingErr)

	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Equal(t, "Entity not found", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
	assert.Equal(t, "Entity not found", err.Error())
}

func TestNewBadRequest(t *testing.T) {
	underlyingErr := errors.New("invalid input")
	err := NewBadRequest("Invalid request data", underlyingErr)

	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "Invalid request data", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
	assert.Equal(t, "Invalid request data", err.Error())
}

func TestNewConflict(t *testing.T) {
	underlyingErr := errors.New("duplicate key")
	err := NewConflict("Resource already exists", underlyingErr)

	assert.Equal(t, http.StatusConflict, err.StatusCode)
	assert.Equal(t, "Resource already exists", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
	assert.Equal(t, "Resource already exists", err.Error())
}

func TestNewInternalServerError(t *testing.T) {
	underlyingErr := errors.New("database connection failed")
	err := NewInternalServerError("Internal error occurred", underlyingErr)

	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.Equal(t, "Internal error occurred", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
	assert.Equal(t, "Internal error occurred", err.Error())
}

func TestErrorConstructorsWithNilError(t *testing.T) {
	tests := []struct {
		name           string
		constructor    func(string, error) *Error
		expectedStatus int
		message        string
	}{
		{
			name:           "NewNotFound with nil error",
			constructor:    NewNotFound,
			expectedStatus: http.StatusNotFound,
			message:        "Not found",
		},
		{
			name:           "NewBadRequest with nil error",
			constructor:    NewBadRequest,
			expectedStatus: http.StatusBadRequest,
			message:        "Bad request",
		},
		{
			name:           "NewConflict with nil error",
			constructor:    NewConflict,
			expectedStatus: http.StatusConflict,
			message:        "Conflict",
		},
		{
			name:           "NewInternalServerError with nil error",
			constructor:    NewInternalServerError,
			expectedStatus: http.StatusInternalServerError,
			message:        "Internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor(tt.message, nil)

			assert.Equal(t, tt.expectedStatus, err.StatusCode)
			assert.Equal(t, tt.message, err.Message)
			assert.Nil(t, err.Err)
			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestError_ErrorsIsCompatibility(t *testing.T) {
	underlyingErr := errors.New("original error")
	httpErr := NewBadRequest("wrapper message", underlyingErr)

	assert.True(t, errors.Is(httpErr, underlyingErr))
	assert.False(t, errors.Is(httpErr, errors.New("different error")))
}

func TestError_ErrorsAsCompatibility(t *testing.T) {
	customErr := &CustomError{Code: 123}
	httpErr := NewBadRequest("wrapper message", customErr)

	var target *CustomError
	assert.True(t, errors.As(httpErr, &target))
	assert.Equal(t, 123, target.Code)
}

func TestError_ChainedWrapping(t *testing.T) {
	originalErr := errors.New("root cause")
	middleErr := NewBadRequest("middle error", originalErr)
	topErr := NewInternalServerError("top error", middleErr)

	assert.Equal(t, "top error", topErr.Error())
	assert.Equal(t, middleErr, topErr.Unwrap())
	assert.True(t, errors.Is(topErr, originalErr))
	assert.True(t, errors.Is(topErr, middleErr))
}

func TestError_AllStatusCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   string
	}{
		{http.StatusOK, "OK"},
		{http.StatusCreated, "Created"},
		{http.StatusBadRequest, "Bad Request"},
		{http.StatusUnauthorized, "Unauthorized"},
		{http.StatusForbidden, "Forbidden"},
		{http.StatusNotFound, "Not Found"},
		{http.StatusConflict, "Conflict"},
		{http.StatusInternalServerError, "Internal Server Error"},
		{http.StatusServiceUnavailable, "Service Unavailable"},
		{999, ""},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			err := &Error{
				StatusCode: tt.statusCode,
				Message:    "",
				Err:        nil,
			}
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}
