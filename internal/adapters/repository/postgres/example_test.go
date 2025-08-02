package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"microservice/internal/adapters/database"
	"microservice/internal/config"
	"microservice/internal/core/domain/example"
	"microservice/internal/platform/logger"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RepositoryTestSuite struct {
	suite.Suite
	db         *database.Lifecycle
	repository *Repository
	pg         *postgres.PostgresContainer
}

func (s *RepositoryTestSuite) SetupSuite() {
	ctx := context.Background()

	pg, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	s.Require().NoError(err)
	s.pg = pg

	host, err := pg.Host(ctx)
	s.Require().NoError(err)
	port, err := pg.MappedPort(ctx, "5432")
	s.Require().NoError(err)

	dbConfig := &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "postgres",
			Password: "postgres",
			Database: "test-db",
			SSLMode:  "disable",
		},
	}

	log := logger.NewNop()

	s.db = database.NewDatabaseLifecycle(dbConfig, log)
	err = s.db.Start(ctx)
	s.Require().NoError(err)

	s.repository = NewRepository(s.db)
	err = s.repository.CreateTable(ctx)
	s.Require().NoError(err)
}

func (s *RepositoryTestSuite) SetupTest() {
	ctx := context.Background()
	_, err := s.db.Connection().ExecContext(ctx, "TRUNCATE TABLE examples")
	s.Require().NoError(err)
}

func (s *RepositoryTestSuite) TearDownSuite() {
	ctx := context.Background()
	err := s.db.Stop(ctx)
	s.Require().NoError(err)
	err = s.pg.Terminate(ctx)
	s.Require().NoError(err)
}

func (s *RepositoryTestSuite) TestSaveAndGetByID() {
	ctx := context.Background()
	entity := &example.Entity{
		ID:    "test-id-123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	err := s.repository.Save(ctx, entity)
	s.Require().NoError(err)

	retrieved, err := s.repository.GetByID(ctx, entity.ID)
	s.Require().NoError(err)
	s.Require().NotNil(retrieved)

	s.Equal(entity.ID, retrieved.ID)
	s.Equal(entity.Email, retrieved.Email)
	s.Equal(entity.Name, retrieved.Name)
}

func (s *RepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()
	retrieved, err := s.repository.GetByID(ctx, "nonexistent-id")
	s.Require().Error(err)
	s.Require().Nil(retrieved)
	s.True(errors.Is(err, example.ErrEntityNotFound))
}

func (s *RepositoryTestSuite) TestSave_AlreadyExists() {
	ctx := context.Background()
	entity := &example.Entity{
		ID:    "duplicate-id-456",
		Email: "test2@example.com",
		Name:  "Test User 2",
	}

	err := s.repository.Save(ctx, entity)
	s.Require().NoError(err)

	err = s.repository.Save(ctx, entity)
	s.Require().Error(err)
	var alreadyExistsErr *example.AlreadyExistsError
	ok := errors.As(err, &alreadyExistsErr)
	s.Require().True(ok)
	s.Equal(entity.ID, alreadyExistsErr.ID)
}

func (s *RepositoryTestSuite) TestSave_MaxLengthFields() {
	ctx := context.Background()

	longString := strings.Repeat("a", 255)
	entity := &example.Entity{
		ID:    "test-max-length",
		Email: fmt.Sprintf("%s@example.com", strings.Repeat("a", 240)),
		Name:  longString,
	}

	err := s.repository.Save(ctx, entity)
	s.Require().NoError(err)

	retrieved, err := s.repository.GetByID(ctx, entity.ID)
	s.Require().NoError(err)
	s.Equal(entity.Email, retrieved.Email)
	s.Equal(entity.Name, retrieved.Name)
}

func (s *RepositoryTestSuite) TestSave_UnicodeCharacters() {
	ctx := context.Background()
	entity := &example.Entity{
		ID:    "test-unicode",
		Email: "тест@пример.рф",
		Name:  "Тестовый Пользователь 测试用户 🚀",
	}

	err := s.repository.Save(ctx, entity)
	s.Require().NoError(err)

	retrieved, err := s.repository.GetByID(ctx, entity.ID)
	s.Require().NoError(err)
	s.Equal(entity.Email, retrieved.Email)
	s.Equal(entity.Name, retrieved.Name)
}

func (s *RepositoryTestSuite) TestSave_SQLInjectionPrevention() {
	ctx := context.Background()
	entity := &example.Entity{
		ID:    "test-sql-injection",
		Email: "test'; DROP TABLE examples; --@example.com",
		Name:  "'; DELETE FROM examples; --",
	}

	err := s.repository.Save(ctx, entity)
	s.Require().NoError(err)

	retrieved, err := s.repository.GetByID(ctx, entity.ID)
	s.Require().NoError(err)
	s.Equal(entity.Email, retrieved.Email)
	s.Equal(entity.Name, retrieved.Name)

	var count int
	err = s.db.Connection().QueryRowContext(ctx, "SELECT COUNT(*) FROM examples").Scan(&count)
	s.Require().NoError(err)
	s.GreaterOrEqual(count, 1)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
