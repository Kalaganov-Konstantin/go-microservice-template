package memory

import (
	"context"
	"sync"
)

type Entity interface {
	GetID() string
}

type Repository[T Entity] struct {
	data map[string]T
	mu   sync.RWMutex
}

func New[T Entity]() *Repository[T] {
	return &Repository[T]{
		data: make(map[string]T),
	}
}

func (r *Repository[T]) Save(ctx context.Context, entity T) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	id := entity.GetID()
	if _, exists := r.data[id]; exists {
		return ErrAlreadyExists
	}

	r.data[id] = entity
	return nil
}

func (r *Repository[T]) GetByID(ctx context.Context, id string) (T, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	var zero T
	entity, exists := r.data[id]
	if !exists {
		return zero, ErrNotFound
	}

	return entity, nil
}

func (r *Repository[T]) Update(ctx context.Context, entity T) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	id := entity.GetID()
	if _, exists := r.data[id]; !exists {
		return ErrNotFound
	}

	r.data[id] = entity
	return nil
}

func (r *Repository[T]) Delete(ctx context.Context, id string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; !exists {
		return ErrNotFound
	}

	delete(r.data, id)
	return nil
}

func (r *Repository[T]) List(ctx context.Context) ([]T, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]T, 0, len(r.data))
	for _, entity := range r.data {
		entities = append(entities, entity)
	}

	return entities, nil
}

func (r *Repository[T]) Count(ctx context.Context) (int, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.data), nil
}
