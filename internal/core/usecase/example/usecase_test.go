package example

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microservice/internal/core/domain/example"
	portsMocks "microservice/internal/core/ports/mocks"
	"microservice/internal/core/usecase/example/mocks"
)

func TestNewUsecase(t *testing.T) {
	mockRepo := portsMocks.NewMockExampleRepository(t)
	mockChecker := mocks.NewMockEntityChecker(t)

	uc := NewUsecase(mockRepo, mockChecker)

	require.NotNil(t, uc, "NewUsecase() should not return nil")
	assert.Equal(t, mockRepo, uc.repo)
	assert.Equal(t, mockChecker, uc.checker)
}

func TestUsecase_GetEntity(t *testing.T) {
	tests := []struct {
		name           string
		entityID       string
		setupMocks     func(*portsMocks.MockExampleRepository)
		expectedEntity *example.Entity
		expectedError  error
	}{
		{
			name:     "successful_get_entity",
			entityID: "test-id",
			setupMocks: func(repo *portsMocks.MockExampleRepository) {
				entity := &example.Entity{
					ID:    "test-id",
					Email: "test@example.com",
					Name:  "Test User",
				}
				repo.EXPECT().GetByID(context.Background(), "test-id").Return(entity, nil).Once()
			},
			expectedEntity: &example.Entity{
				ID:    "test-id",
				Email: "test@example.com",
				Name:  "Test User",
			},
			expectedError: nil,
		},
		{
			name:     "entity_not_found",
			entityID: "nonexistent-id",
			setupMocks: func(repo *portsMocks.MockExampleRepository) {
				repo.EXPECT().GetByID(context.Background(), "nonexistent-id").Return(nil, example.ErrEntityNotFound).Once()
			},
			expectedEntity: nil,
			expectedError:  example.ErrEntityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := portsMocks.NewMockExampleRepository(t)
			mockService := mocks.NewMockEntityChecker(t)

			tt.setupMocks(mockRepo)

			uc := NewUsecase(mockRepo, mockService)
			ctx := context.Background()

			entity, err := uc.GetEntity(ctx, tt.entityID)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, entity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entity)
				assert.Equal(t, tt.expectedEntity.ID, entity.ID)
				assert.Equal(t, tt.expectedEntity.Email, entity.Email)
				assert.Equal(t, tt.expectedEntity.Name, entity.Name)
			}

			mockRepo.AssertExpectations(t)
			mockService.AssertExpectations(t)
		})
	}
}

func TestUsecase_CreateEntity(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		email         string
		entityName    string
		setupMocks    func(*portsMocks.MockExampleRepository, *mocks.MockEntityChecker)
		expectedError error
	}{
		{
			name:       "successful_creation",
			id:         "test-id",
			email:      "test@example.com",
			entityName: "Test User",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				service.EXPECT().CheckEntityForCreation(&example.Entity{
					ID:    "test-id",
					Email: "test@example.com",
					Name:  "Test User",
				}).Return(nil).Once()

				repo.EXPECT().Save(context.Background(), &example.Entity{
					ID:    "test-id",
					Email: "test@example.com",
					Name:  "Test User",
				}).Return(nil).Once()
			},
			expectedError: nil,
		},
		{
			name:       "invalid_entity_id",
			id:         "",
			email:      "test@example.com",
			entityName: "Test User",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				// No mock setup needed as validation happens before repository/service calls
			},
			expectedError: example.ErrInvalidEntityID,
		},
		{
			name:       "invalid_email",
			id:         "test-id",
			email:      "invalid-email",
			entityName: "Test User",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				// No mock setup needed as validation happens before repository/service calls
			},
			expectedError: example.ErrInvalidEmail,
		},
		{
			name:       "invalid_name",
			id:         "test-id",
			email:      "test@example.com",
			entityName: "",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				// No mock setup needed as validation happens before repository/service calls
			},
			expectedError: example.ErrInvalidName,
		},
		{
			name:       "service_check_failed_reserved_name",
			id:         "test-id",
			email:      "admin@example.com",
			entityName: "admin",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				service.EXPECT().CheckEntityForCreation(&example.Entity{
					ID:    "test-id",
					Email: "admin@example.com",
					Name:  "admin",
				}).Return(example.ErrReservedName).Once()
			},
			expectedError: example.ErrReservedName,
		},
		{
			name:       "entity_already_exists",
			id:         "existing-id",
			email:      "test@example.com",
			entityName: "Test User",
			setupMocks: func(repo *portsMocks.MockExampleRepository, service *mocks.MockEntityChecker) {
				service.EXPECT().CheckEntityForCreation(&example.Entity{
					ID:    "existing-id",
					Email: "test@example.com",
					Name:  "Test User",
				}).Return(nil).Once()

				repo.EXPECT().Save(context.Background(), &example.Entity{
					ID:    "existing-id",
					Email: "test@example.com",
					Name:  "Test User",
				}).Return(&example.AlreadyExistsError{ID: "existing-id"}).Once()
			},
			expectedError: &example.AlreadyExistsError{ID: "existing-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := portsMocks.NewMockExampleRepository(t)
			mockService := mocks.NewMockEntityChecker(t)

			tt.setupMocks(mockRepo, mockService)

			uc := NewUsecase(mockRepo, mockService)
			ctx := context.Background()

			entity, err := uc.CreateEntity(ctx, tt.id, tt.email, tt.entityName)

			if tt.expectedError != nil {
				require.Error(t, err)

				var alreadyExistsErr *example.AlreadyExistsError
				var expectedAlreadyExistsErr *example.AlreadyExistsError
				if errors.As(tt.expectedError, &expectedAlreadyExistsErr) {
					require.ErrorAs(t, err, &alreadyExistsErr)
					assert.Equal(t, expectedAlreadyExistsErr.ID, alreadyExistsErr.ID)
				}

				assert.Nil(t, entity)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entity)
				assert.Equal(t, tt.id, entity.ID)
				assert.Equal(t, tt.email, entity.Email)
				assert.Equal(t, tt.entityName, entity.Name)
			}

			mockRepo.AssertExpectations(t)
			mockService.AssertExpectations(t)
		})
	}
}
