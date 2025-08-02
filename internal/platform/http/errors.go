package http

import (
	"net/http"
)

type Error struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return http.StatusText(e.StatusCode)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(statusCode int, message string, err error) *Error {
	return &Error{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

func NewNotFound(message string, err error) *Error {
	return New(http.StatusNotFound, message, err)
}

func NewBadRequest(message string, err error) *Error {
	return New(http.StatusBadRequest, message, err)
}

func NewConflict(message string, err error) *Error {
	return New(http.StatusConflict, message, err)
}

func NewInternalServerError(message string, err error) *Error {
	return New(http.StatusInternalServerError, message, err)
}
