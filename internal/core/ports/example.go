package ports

import (
	"context"
	"microservice/internal/core/domain/example"
)

type ExampleRepository interface {
	Save(ctx context.Context, entity *example.Entity) error
	GetByID(ctx context.Context, id string) (*example.Entity, error)
}
