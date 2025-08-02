package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CheckEntityForCreation(t *testing.T) {
	service := NewService()

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
			name:       "reserved name - admin lowercase",
			id:         "test-id",
			email:      "admin@example.com",
			entityName: "admin",
			wantErr:    ErrReservedName,
		},
		{
			name:       "reserved name - Admin mixed case",
			id:         "test-id",
			email:      "admin@example.com",
			entityName: "Admin",
			wantErr:    ErrReservedName,
		},
		{
			name:       "reserved name - ADMIN uppercase",
			id:         "test-id",
			email:      "admin@example.com",
			entityName: "ADMIN",
			wantErr:    ErrReservedName,
		},
		{
			name:       "valid name containing admin",
			id:         "test-id",
			email:      "test@example.com",
			entityName: "administrator",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := NewEntity(tt.id, tt.email, tt.entityName)
			require.NoError(t, err, "NewEntity should not fail in test setup")

			err = service.CheckEntityForCreation(entity)

			if tt.wantErr != nil {
				require.Error(t, err, "CheckEntityForCreation() should return error")
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err, "CheckEntityForCreation() should not return error")
		})
	}
}

func TestNewService(t *testing.T) {
	service := NewService()

	require.NotNil(t, service, "NewService() should not return nil")
}
