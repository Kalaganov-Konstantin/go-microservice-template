package example

import (
	"context"

	"microservice/internal/core/domain/example"
)

type Manager interface {
	GetEntity(ctx context.Context, id string) (*example.Entity, error)
	CreateEntity(ctx context.Context, id, email, name string) (*example.Entity, error)
}
