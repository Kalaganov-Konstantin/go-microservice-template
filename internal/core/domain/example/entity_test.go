package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntity(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		email      string
		entityName string
		wantErr    error
	}{
		{
			name:       "valid entity",
			id:         "test-id",
			email:      "test@example.com",
			entityName: "Test User",
			wantErr:    nil,
		},
		{
			name:       "empty id",
			id:         "",
			email:      "test@example.com",
			entityName: "Test User",
			wantErr:    ErrInvalidEntityID,
		},
		{
			name:       "empty name",
			id:         "test-id",
			email:      "test@example.com",
			entityName: "",
			wantErr:    ErrInvalidName,
		},
		{
			name:       "invalid email",
			id:         "test-id",
			email:      "invalid-email",
			entityName: "Test User",
			wantErr:    ErrInvalidEmail,
		},
		{
			name:       "email without domain",
			id:         "test-id",
			email:      "test@",
			entityName: "Test User",
			wantErr:    ErrInvalidEmail,
		},
		{
			name:       "email without @",
			id:         "test-id",
			email:      "testexample.com",
			entityName: "Test User",
			wantErr:    ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := NewEntity(tt.id, tt.email, tt.entityName)

			if tt.wantErr != nil {
				require.Error(t, err, "NewEntity() should return error")
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, entity, "entity should be nil when error occurs")
				return
			}

			require.NoError(t, err, "NewEntity() should not return error")
			require.NotNil(t, entity, "entity should not be nil")

			assert.Equal(t, tt.id, entity.ID)
			assert.Equal(t, tt.email, entity.Email)
			assert.Equal(t, tt.entityName, entity.Name)
		})
	}
}

func TestEntity_GetID(t *testing.T) {
	entity, err := NewEntity("test-id", "test@example.com", "Test User")
	assert.NoError(t, err)
	result := entity.GetID()
	assert.Equal(t, "test-id", result)
}

func TestAlreadyExistsError_Error(t *testing.T) {
	err := &AlreadyExistsError{ID: "test-id"}
	expected := "entity with id 'test-id' already exists"

	result := err.Error()
	assert.Equal(t, expected, result)
}
