package example

import "microservice/internal/core/domain/example"

type EntityChecker interface {
	CheckEntityForCreation(entity *example.Entity) error
}
