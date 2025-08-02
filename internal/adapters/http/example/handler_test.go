package example

import (
	"bytes"
	"encoding/json"
	"errors"
	"microservice/internal/adapters/http/example/mocks"
	"microservice/internal/adapters/http/response"
	"microservice/internal/core/domain/example"
	httpErrors "microservice/internal/platform/http"
	"microservice/internal/platform/logger"
	"microservice/internal/platform/validator"
	validatorMocks "microservice/internal/platform/validator/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite
	mockManager   *mocks.MockManager
	mockValidator *validatorMocks.MockValidator
	handler       *Handler
	router        *chi.Mux
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.mockManager = mocks.NewMockManager(suite.T())
	suite.mockValidator = validatorMocks.NewMockValidator(suite.T())
	suite.handler = NewHandler(suite.mockManager, suite.mockValidator)

	suite.router = chi.NewRouter()
	suite.router.Get("/entities/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := suite.handler.GetEntity(w, r)
		if err != nil {
			var httpErr *httpErrors.Error
			if errors.As(err, &httpErr) {
				response.RespondError(w, httpErr.StatusCode, httpErr)
			} else {
				response.RespondError(w, http.StatusInternalServerError, err)
			}
		}
	})

	suite.router.Post("/entities", func(w http.ResponseWriter, r *http.Request) {
		err := suite.handler.CreateEntity(w, r)
		if err != nil {
			var httpErr *httpErrors.Error
			if errors.As(err, &httpErr) {
				response.RespondError(w, httpErr.StatusCode, httpErr)
			} else {
				response.RespondError(w, http.StatusInternalServerError, err)
			}
		}
	})
}

func (suite *HandlerTestSuite) TestGetEntity_Success() {
	expectedEntity := &example.Entity{
		ID:    "test-id",
		Email: "test@example.com",
		Name:  "Test Name",
	}

	suite.mockManager.EXPECT().
		GetEntity(mock.Anything, "test-id").
		Return(expectedEntity, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/entities/test-id", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var responseEntity example.Entity
	err := json.Unmarshal(w.Body.Bytes(), &responseEntity)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedEntity.ID, responseEntity.ID)
	assert.Equal(suite.T(), expectedEntity.Email, responseEntity.Email)
	assert.Equal(suite.T(), expectedEntity.Name, responseEntity.Name)
}

func (suite *HandlerTestSuite) TestGetEntity_NotFound() {
	suite.mockManager.EXPECT().
		GetEntity(mock.Anything, "nonexistent-id").
		Return(nil, example.ErrEntityNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/entities/nonexistent-id", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.JSONEq(suite.T(), `{"error":"Entity not found"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestGetEntity_InvalidEntityID() {
	suite.mockManager.EXPECT().
		GetEntity(mock.Anything, "invalid-id").
		Return(nil, example.ErrInvalidEntityID).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/entities/invalid-id", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.JSONEq(suite.T(), `{"error":"Invalid entity ID"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestCreateEntity_Success() {
	request := CreateEntityRequest{
		ID:    "test-id",
		Email: "test@example.com",
		Name:  "Test Name",
	}

	expectedEntity := &example.Entity{
		ID:    "test-id",
		Email: "test@example.com",
		Name:  "Test Name",
	}

	suite.mockValidator.EXPECT().
		Validate(request).
		Return(nil).
		Once()

	suite.mockManager.EXPECT().
		CreateEntity(mock.Anything, "test-id", "test@example.com", "Test Name").
		Return(expectedEntity, nil).
		Once()

	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBuffer(body))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var responseEntity example.Entity
	err = json.Unmarshal(w.Body.Bytes(), &responseEntity)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedEntity.ID, responseEntity.ID)
	assert.Equal(suite.T(), expectedEntity.Email, responseEntity.Email)
	assert.Equal(suite.T(), expectedEntity.Name, responseEntity.Name)
}

func (suite *HandlerTestSuite) TestCreateEntity_InvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBufferString("invalid json"))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.JSONEq(suite.T(), `{"error":"invalid request payload"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestCreateEntity_ValidationError() {
	request := CreateEntityRequest{
		ID:    "",
		Email: "invalid-email",
		Name:  "",
	}

	validationErr := validator.ValidationError{
		Errors: []validator.FieldError{
			{Field: "id", Message: "required"},
			{Field: "email", Message: "invalid format"},
			{Field: "name", Message: "required"},
		},
	}

	suite.mockValidator.EXPECT().
		Validate(request).
		Return(validationErr).
		Once()

	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBuffer(body))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var validationResponse validator.ValidationError
	err = json.Unmarshal(w.Body.Bytes(), &validationResponse)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), validationResponse.Errors, 3)
}

func (suite *HandlerTestSuite) TestCreateEntity_EntityAlreadyExists() {
	request := CreateEntityRequest{
		ID:    "existing-id",
		Email: "test@example.com",
		Name:  "Test Name",
	}

	suite.mockValidator.EXPECT().
		Validate(request).
		Return(nil).
		Once()

	suite.mockManager.EXPECT().
		CreateEntity(mock.Anything, "existing-id", "test@example.com", "Test Name").
		Return(nil, &example.AlreadyExistsError{ID: "existing-id"}).
		Once()

	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBuffer(body))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusConflict, w.Code)
	assert.JSONEq(suite.T(), `{"error":"Entity already exists"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestCreateEntity_InvalidEmail() {
	request := CreateEntityRequest{
		ID:    "test-id",
		Email: "invalid-email",
		Name:  "Test Name",
	}

	suite.mockValidator.EXPECT().
		Validate(request).
		Return(nil).
		Once()

	suite.mockManager.EXPECT().
		CreateEntity(mock.Anything, "test-id", "invalid-email", "Test Name").
		Return(nil, example.ErrInvalidEmail).
		Once()

	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBuffer(body))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.JSONEq(suite.T(), `{"error":"Invalid email format"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestCreateEntity_ReservedName() {
	request := CreateEntityRequest{
		ID:    "test-id",
		Email: "test@example.com",
		Name:  "admin",
	}

	suite.mockValidator.EXPECT().
		Validate(request).
		Return(nil).
		Once()

	suite.mockManager.EXPECT().
		CreateEntity(mock.Anything, "test-id", "test@example.com", "admin").
		Return(nil, example.ErrReservedName).
		Once()

	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewBuffer(body))
	req = req.WithContext(logger.WithLogger(req.Context(), logger.NewNop()))
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.JSONEq(suite.T(), `{"error":"Name is reserved"}`, w.Body.String())
}

func (suite *HandlerTestSuite) TestMapDomainError() {
	tests := []struct {
		name           string
		inputError     error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "entity not found error",
			inputError:     example.ErrEntityNotFound,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Entity not found",
		},
		{
			name:           "invalid entity ID error",
			inputError:     example.ErrInvalidEntityID,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid entity ID",
		},
		{
			name:           "invalid email error",
			inputError:     example.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid email format",
		},
		{
			name:           "invalid name error",
			inputError:     example.ErrInvalidName,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid name",
		},
		{
			name:           "reserved name error",
			inputError:     example.ErrReservedName,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Name is reserved",
		},
		{
			name:           "already exists error",
			inputError:     &example.AlreadyExistsError{ID: "test-id"},
			expectedStatus: http.StatusConflict,
			expectedMsg:    "Entity already exists",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := suite.handler.mapDomainError(tt.inputError)

			var httpErr *httpErrors.Error
			ok := errors.As(result, &httpErr)
			require.True(suite.T(), ok, "Expected HTTP error but got %T", result)
			assert.Equal(suite.T(), tt.expectedStatus, httpErr.StatusCode)
			assert.Equal(suite.T(), tt.expectedMsg, httpErr.Message)
		})
	}
}

func (suite *HandlerTestSuite) TestMapDomainError_UnknownError() {
	unknownErr := errors.New("unknown error")
	result := suite.handler.mapDomainError(unknownErr)
	assert.Equal(suite.T(), unknownErr, result)
}

func (suite *HandlerTestSuite) TestNewHandler() {
	handler := NewHandler(suite.mockManager, suite.mockValidator)

	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), suite.mockManager, handler.manager)
	assert.Equal(suite.T(), suite.mockValidator, handler.validate)
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
