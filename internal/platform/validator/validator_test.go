package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldError_Error(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		message  string
		expected string
	}{
		{
			name:     "standard field error",
			field:    "email",
			message:  "is required",
			expected: "field email: is required",
		},
		{
			name:     "empty field name",
			field:    "",
			message:  "invalid value",
			expected: "field : invalid value",
		},
		{
			name:     "empty message",
			field:    "name",
			message:  "",
			expected: "field name: ",
		},
		{
			name:     "complex field name",
			field:    "user.profile.email",
			message:  "must be valid email address",
			expected: "field user.profile.email: must be valid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := FieldError{
				Field:   tt.field,
				Message: tt.message,
			}

			result := fe.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []FieldError
		expected string
	}{
		{
			name:     "single error",
			errors:   []FieldError{{Field: "email", Message: "is required"}},
			expected: "validation failed: field email: is required",
		},
		{
			name: "multiple errors",
			errors: []FieldError{
				{Field: "email", Message: "is required"},
				{Field: "name", Message: "must be at least 2 characters"},
			},
			expected: "validation failed: field email: is required, field name: must be at least 2 characters",
		},
		{
			name:     "empty errors slice",
			errors:   []FieldError{},
			expected: "validation failed: ",
		},
		{
			name: "three errors",
			errors: []FieldError{
				{Field: "id", Message: "is required"},
				{Field: "email", Message: "invalid format"},
				{Field: "name", Message: "too short"},
			},
			expected: "validation failed: field id: is required, field email: invalid format, field name: too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := ValidationError{
				Errors: tt.errors,
			}

			result := ve.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError_ErrorInterface(t *testing.T) {
	ve := ValidationError{
		Errors: []FieldError{
			{Field: "test", Message: "error"},
		},
	}

	var err error = ve
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestFieldError_ErrorInterface(t *testing.T) {
	fe := FieldError{
		Field:   "test",
		Message: "error",
	}

	var err error = fe
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "field test: error")
}

func TestValidationError_EmptyState(t *testing.T) {
	ve := ValidationError{}
	result := ve.Error()
	assert.Equal(t, "validation failed: ", result)
}

func TestFieldError_EmptyState(t *testing.T) {
	fe := FieldError{}
	result := fe.Error()
	assert.Equal(t, "field : ", result)
}

func TestValidationError_StructInitialization(t *testing.T) {
	ve := ValidationError{
		Errors: []FieldError{
			{Field: "email", Message: "required"},
			{Field: "password", Message: "too weak"},
		},
	}

	assert.Len(t, ve.Errors, 2)
	assert.Equal(t, "email", ve.Errors[0].Field)
	assert.Equal(t, "required", ve.Errors[0].Message)
	assert.Equal(t, "password", ve.Errors[1].Field)
	assert.Equal(t, "too weak", ve.Errors[1].Message)
}

func TestValidationError_AppendErrors(t *testing.T) {
	ve := ValidationError{}

	ve.Errors = append(ve.Errors, FieldError{Field: "first", Message: "error1"})
	ve.Errors = append(ve.Errors, FieldError{Field: "second", Message: "error2"})

	assert.Len(t, ve.Errors, 2)
	result := ve.Error()
	assert.Contains(t, result, "field first: error1")
	assert.Contains(t, result, "field second: error2")
}

func TestValidatorInterface_CompileTimeCheck(t *testing.T) {
	var _ Validator = (*mockValidator)(nil)
}

type mockValidator struct {
	returnError error
}

func (mv *mockValidator) Validate(interface{}) error {
	return mv.returnError
}

func TestMockValidator_InterfaceCompliance(t *testing.T) {
	mv := &mockValidator{returnError: nil}

	err := mv.Validate("test")
	assert.NoError(t, err)

	testError := ValidationError{
		Errors: []FieldError{{Field: "test", Message: "invalid"}},
	}
	mv.returnError = testError

	err = mv.Validate("test")
	assert.Error(t, err)
	assert.Equal(t, testError, err)
}
