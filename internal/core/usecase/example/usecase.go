package example

import (
	"context"
	"microservice/internal/platform/logger"

	"microservice/internal/core/domain/example"
	"microservice/internal/core/ports"
)

type Usecase struct {
	repo    ports.ExampleRepository
	checker EntityChecker
}

func NewUsecase(repo ports.ExampleRepository, checker EntityChecker) *Usecase {
	return &Usecase{
		repo:    repo,
		checker: checker,
	}
}

func (uc *Usecase) GetEntity(ctx context.Context, id string) (*example.Entity, error) {
	log := logger.FromContext(ctx)
	log.Debug("Getting entity", logger.String("entity_id", id))

	entity, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (uc *Usecase) CreateEntity(ctx context.Context, id, email, name string) (*example.Entity, error) {
	log := logger.FromContext(ctx)
	log.Debug("Creating entity", logger.String("entity_id", id), logger.String("email", email))

	entity, err := example.NewEntity(id, email, name)
	if err != nil {
		log.Warn("Invalid entity data provided", logger.String("entity_id", id), logger.Error(err))
		return nil, err
	}

	if err := uc.checker.CheckEntityForCreation(entity); err != nil {
		log.Warn("Entity creation check failed", logger.String("entity_id", id), logger.Error(err))
		return nil, err
	}

	if err := uc.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}
