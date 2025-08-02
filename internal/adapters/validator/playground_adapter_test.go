package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	validatorPlatform "microservice/internal/platform/validator"
)

type TestUser struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=0,max=120"`
}

type TestEmpty struct{}

func TestNewPlaygroundAdapter(t *testing.T) {
	validator := NewPlaygroundAdapter()

	require.NotNil(t, validator)
	assert.Implements(t, (*validatorPlatform.Validator)(nil), validator)
}

func TestPlaygroundValidator_Validate_Success(t *testing.T) {
	validator := NewPlaygroundAdapter()

	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	err := validator.Validate(user)

	assert.NoError(t, err)
}

func TestPlaygroundValidator_Validate_EmptyStruct(t *testing.T) {
	validator := NewPlaygroundAdapter()

	empty := TestEmpty{}

	err := validator.Validate(empty)

	assert.NoError(t, err)
}

func TestPlaygroundValidator_Validate_RequiredFieldMissing(t *testing.T) {
	validator := NewPlaygroundAdapter()

	user := TestUser{
		Email: "john@example.com",
		Age:   25,
		// Name is missing
	}

	err := validator.Validate(user)

	require.Error(t, err)

	var validationErr validatorPlatform.ValidationError
	require.ErrorAs(t, err, &validationErr)

	assert.Len(t, validationErr.Errors, 1)
	assert.Equal(t, "name", validationErr.Errors[0].Field)
	assert.Equal(t, "This field is required", validationErr.Errors[0].Message)
}

func TestPlaygroundValidator_Validate_InvalidEmail(t *testing.T) {
	validator := NewPlaygroundAdapter()

	user := TestUser{
		Name:  "John Doe",
		Email: "invalid-email",
		Age:   25,
	}

	err := validator.Validate(user)

	require.Error(t, err)

	var validationErr validatorPlatform.ValidationError
	require.ErrorAs(t, err, &validationErr)

	assert.Len(t, validationErr.Errors, 1)
	assert.Equal(t, "email", validationErr.Errors[0].Field)
	assert.Equal(t, "This field must be a valid email address", validationErr.Errors[0].Message)
}

func TestPlaygroundValidator_Validate_MultipleErrors(t *testing.T) {
	validator := NewPlaygroundAdapter()

	user := TestUser{
		// Name is missing
		Email: "invalid-email",
		Age:   25,
	}

	err := validator.Validate(user)

	require.Error(t, err)

	var validationErr validatorPlatform.ValidationError
	require.ErrorAs(t, err, &validationErr)

	assert.Len(t, validationErr.Errors, 2)

	fields := make(map[string]string)
	for _, fieldErr := range validationErr.Errors {
		fields[fieldErr.Field] = fieldErr.Message
	}

	assert.Equal(t, "This field is required", fields["name"])
	assert.Equal(t, "This field must be a valid email address", fields["email"])
}

func TestPlaygroundValidator_Validate_UnknownTag(t *testing.T) {
	validator := NewPlaygroundAdapter()

	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   -1, // Invalid age (min=0)
	}

	err := validator.Validate(user)

	require.Error(t, err)

	var validationErr validatorPlatform.ValidationError
	require.ErrorAs(t, err, &validationErr)

	assert.Len(t, validationErr.Errors, 1)
	assert.Equal(t, "age", validationErr.Errors[0].Field)
	assert.Contains(t, validationErr.Errors[0].Message, "This field failed on the 'min' tag")
}

func TestPlaygroundValidator_Validate_NonStructError(t *testing.T) {
	validator := NewPlaygroundAdapter()

	// Test with a non-struct type (should return error but not ValidationError)
	err := validator.Validate("not a struct")

	require.Error(t, err)

	var validationErr validatorPlatform.ValidationError
	assert.False(t, errors.As(err, &validationErr))
}

func TestGetValidationErrorMessage(t *testing.T) {
	testCases := []struct {
		name            string
		user            TestUser
		expectedField   string
		expectedMessage string
	}{
		{
			name: "required tag",
			user: TestUser{
				Email: "john@example.com",
				Age:   25,
			},
			expectedField:   "name",
			expectedMessage: "This field is required",
		},
		{
			name: "email tag",
			user: TestUser{
				Name:  "John",
				Email: "invalid",
				Age:   25,
			},
			expectedField:   "email",
			expectedMessage: "This field must be a valid email address",
		},
	}

	validator := NewPlaygroundAdapter()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.user)

			require.Error(t, err)

			var validationErr validatorPlatform.ValidationError
			require.ErrorAs(t, err, &validationErr)

			found := false
			for _, fieldErr := range validationErr.Errors {
				if fieldErr.Field == tc.expectedField {
					assert.Equal(t, tc.expectedMessage, fieldErr.Message)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected field error not found")
		})
	}
}

func BenchmarkPlaygroundValidator_Validate_Success(b *testing.B) {
	validator := NewPlaygroundAdapter()
	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user)
	}
}

func BenchmarkPlaygroundValidator_Validate_WithErrors(b *testing.B) {
	validator := NewPlaygroundAdapter()
	user := TestUser{
		Email: "invalid-email",
		Age:   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user)
	}
}
