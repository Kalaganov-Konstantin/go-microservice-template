package memory

import (
	"context"
	"errors"
	memoryPlatform "microservice/internal/platform/repository/memory"

	"microservice/internal/core/domain/example"
)

type Repository struct {
	*memoryPlatform.Repository[*example.Entity]
}

func NewRepository() *Repository {
	return &Repository{
		Repository: memoryPlatform.New[*example.Entity](),
	}
}

func (r *Repository) GetByID(ctx context.Context, id string) (*example.Entity, error) {
	entity, err := r.Repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, memoryPlatform.ErrNotFound) {
			return nil, example.ErrEntityNotFound
		}
		return nil, err
	}
	return entity, nil
}

func (r *Repository) Save(ctx context.Context, entity *example.Entity) error {
	err := r.Repository.Save(ctx, entity)
	if err != nil {
		if errors.Is(err, memoryPlatform.ErrAlreadyExists) {
			return &example.AlreadyExistsError{ID: entity.ID}
		}
		return err
	}
	return nil
}
