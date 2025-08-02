package example

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrInvalidEntityID = errors.New("entity ID cannot be empty")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidName     = errors.New("name cannot be empty")
	emailRegex         = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	ErrEntityNotFound  = errors.New("entity not found")
)

type AlreadyExistsError struct {
	ID string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entity with id '%s' already exists", e.ID)
}

type Entity struct {
	ID    string
	Email string
	Name  string
}

func (e *Entity) GetID() string {
	return e.ID
}

func NewEntity(id, email, name string) (*Entity, error) {
	if id == "" {
		return nil, ErrInvalidEntityID
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}
	return &Entity{
		ID:    id,
		Email: email,
		Name:  name,
	}, nil
}
