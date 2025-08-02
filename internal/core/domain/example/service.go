package example

import (
	"errors"
	"strings"
)

var (
	ErrReservedName = errors.New("name is reserved")
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CheckEntityForCreation(entity *Entity) error {
	if strings.ToLower(entity.Name) == "admin" {
		return ErrReservedName
	}
	return nil
}
