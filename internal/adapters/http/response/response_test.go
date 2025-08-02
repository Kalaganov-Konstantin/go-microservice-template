package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondJSON_Success(t *testing.T) {
	payload := map[string]string{"message": "success"}
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusOK, payload)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["message"])
}

func TestRespondJSON_WithStruct(t *testing.T) {
	type TestStruct struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	payload := TestStruct{ID: 1, Name: "test"}
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusCreated, payload)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response TestStruct
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.ID)
	assert.Equal(t, "test", response.Name)
}

func TestRespondJSON_WithSlice(t *testing.T) {
	payload := []string{"item1", "item2", "item3"}
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusOK, payload)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 3, len(response))
	assert.Equal(t, "item1", response[0])
}

func TestRespondJSON_WithNil(t *testing.T) {
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusNoContent, nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "null\n", w.Body.String())
}

func TestRespondJSON_WithUnencodablePayload(t *testing.T) {
	payload := map[string]interface{}{
		"func": func() {},
	}
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusInternalServerError, payload)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal server error")
}

func TestRespondError_BasicError(t *testing.T) {
	err := errors.New("test error message")
	w := httptest.NewRecorder()

	RespondError(w, http.StatusBadRequest, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	jsonErr := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, jsonErr)
	assert.Equal(t, "test error message", response["error"])
}

func TestRespondError_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		error      error
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			error:      errors.New("invalid input"),
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			error:      errors.New("resource not found"),
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			error:      errors.New("something went wrong"),
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			error:      errors.New("access denied"),
		},
		{
			name:       "conflict",
			statusCode: http.StatusConflict,
			error:      errors.New("resource already exists"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			RespondError(w, tt.statusCode, tt.error)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.error.Error(), response["error"])
		})
	}
}

func TestFieldError_Struct(t *testing.T) {
	fieldErr := FieldError{
		Field:   "email",
		Message: "invalid format",
	}

	assert.Equal(t, "email", fieldErr.Field)
	assert.Equal(t, "invalid format", fieldErr.Message)
}

func TestValidationErrorResponse_JSON(t *testing.T) {
	validationErr := ValidationErrorResponse{
		Errors: []FieldError{
			{Field: "email", Message: "required"},
			{Field: "name", Message: "too short"},
		},
	}
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusBadRequest, validationErr)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ValidationErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Errors, 2)
	assert.Equal(t, "email", response.Errors[0].Field)
	assert.Equal(t, "required", response.Errors[0].Message)
	assert.Equal(t, "name", response.Errors[1].Field)
	assert.Equal(t, "too short", response.Errors[1].Message)
}

func TestRespondJSON_HeadersNotOverwritten(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Custom-Header", "custom-value")
	payload := map[string]string{"test": "data"}

	RespondJSON(w, http.StatusOK, payload)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
}
