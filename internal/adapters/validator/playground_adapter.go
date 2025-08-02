package validator

import (
	"errors"
	"fmt"
	validatorPLatform "microservice/internal/platform/validator"
	"strings"

	"github.com/go-playground/validator/v10"
)

type playgroundValidator struct {
	validate *validator.Validate
}

func NewPlaygroundAdapter() validatorPLatform.Validator {
	return &playgroundValidator{
		validate: validator.New(),
	}
}

func (v *playgroundValidator) Validate(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			outErrors := make([]validatorPLatform.FieldError, len(validationErrors))
			for i, fe := range validationErrors {
				outErrors[i] = validatorPLatform.FieldError{
					Field:   strings.ToLower(fe.Field()),
					Message: getValidationErrorMessage(fe),
				}
			}
			return validatorPLatform.ValidationError{Errors: outErrors}
		}
		return err
	}
	return nil
}

func getValidationErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "This field must be a valid email address"
	default:
		return fmt.Sprintf("This field failed on the '%s' tag", e.Tag())
	}
}
