package validator

import (
	"fmt"
	"strings"
)

type FieldError struct {
	Field   string
	Message string
}

func (fe FieldError) Error() string {
	return fmt.Sprintf("field %s: %s", fe.Field, fe.Message)
}

type ValidationError struct {
	Errors []FieldError
}

func (ve ValidationError) Error() string {
	var errs []string
	for _, fe := range ve.Errors {
		errs = append(errs, fe.Error())
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(errs, ", "))
}

type Validator interface {
	Validate(s interface{}) error
}
