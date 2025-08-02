package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microservice/internal/core/domain/example"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository()

	require.NotNil(t, repo, "NewRepository() should not return nil")
	require.NotNil(t, repo.Repository, "Repository should have embedded memory repository")
}

func TestRepository_Save(t *testing.T) {
	tests := []struct {
		name          string
		entity        *example.Entity
		setupRepo     func(*Repository)
		expectedError error
	}{
		{
			name: "successful_save",
			entity: &example.Entity{
				ID:    "test-id",
				Email: "test@example.com",
				Name:  "Test User",
			},
			setupRepo:     func(repo *Repository) {},
			expectedError: nil,
		},
		{
			name: "entity_already_exists",
			entity: &example.Entity{
				ID:    "existing-id",
				Email: "test@example.com",
				Name:  "Test User",
			},
			setupRepo: func(repo *Repository) {
				existingEntity := &example.Entity{
					ID:    "existing-id",
					Email: "existing@example.com",
					Name:  "Existing User",
				}
				ctx := context.Background()
				_ = repo.Repository.Save(ctx, existingEntity)
			},
			expectedError: &example.AlreadyExistsError{ID: "existing-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository()
			tt.setupRepo(repo)

			ctx := context.Background()
			err := repo.Save(ctx, tt.entity)

			if tt.expectedError != nil {
				require.Error(t, err)

				var alreadyExistsErr *example.AlreadyExistsError
				var expectedAlreadyExistsErr *example.AlreadyExistsError
				if errors.As(tt.expectedError, &expectedAlreadyExistsErr) {
					require.ErrorAs(t, err, &alreadyExistsErr)
					assert.Equal(t, expectedAlreadyExistsErr.ID, alreadyExistsErr.ID)
				}
			} else {
				require.NoError(t, err)

				savedEntity, getErr := repo.GetByID(ctx, tt.entity.ID)
				require.NoError(t, getErr)
				assert.Equal(t, tt.entity.ID, savedEntity.ID)
				assert.Equal(t, tt.entity.Email, savedEntity.Email)
				assert.Equal(t, tt.entity.Name, savedEntity.Name)
			}
		})
	}
}

func TestRepository_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		entityID       string
		setupRepo      func(*Repository)
		expectedEntity *example.Entity
		expectedError  error
	}{
		{
			name:     "successful_get",
			entityID: "test-id",
			setupRepo: func(repo *Repository) {
				entity := &example.Entity{
					ID:    "test-id",
					Email: "test@example.com",
					Name:  "Test User",
				}
				ctx := context.Background()
				_ = repo.Repository.Save(ctx, entity)
			},
			expectedEntity: &example.Entity{
				ID:    "test-id",
				Email: "test@example.com",
				Name:  "Test User",
			},
			expectedError: nil,
		},
		{
			name:           "entity_not_found",
			entityID:       "nonexistent-id",
			setupRepo:      func(repo *Repository) {},
			expectedEntity: nil,
			expectedError:  example.ErrEntityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository()
			tt.setupRepo(repo)

			ctx := context.Background()
			entity, err := repo.GetByID(ctx, tt.entityID)

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
		})
	}
}
