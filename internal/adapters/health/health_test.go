//go:build integration
// +build integration

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"microservice/internal/adapters/database"
	"microservice/internal/config"
	"microservice/internal/platform/health"
	"microservice/internal/platform/logger"
)

func TestNewDatabaseChecker(t *testing.T) {
	cfg := &config.DatabaseConfig{}
	log := logger.NewNop()
	db := database.NewDatabaseLifecycle(cfg, log)

	checker := NewDatabaseChecker(db, "test-db")

	require.NotNil(t, checker)
	assert.Equal(t, "test-db", checker.Name())
	assert.Equal(t, db, checker.db)
}

func TestDatabaseChecker_Check_NoConnection(t *testing.T) {
	cfg := &config.DatabaseConfig{}
	log := logger.NewNop()
	db := database.NewDatabaseLifecycle(cfg, log)
	checker := NewDatabaseChecker(db, "test-db")

	ctx := context.Background()
	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, "database connection is not initialized", result.Message)
	assert.Empty(t, result.Error)
}

func TestDatabaseChecker_Check_ConnectionFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     1234,
			User:     "invalid",
			Password: "invalid",
			Database: "invalid",
		},
	}
	log := logger.NewNop()
	db := database.NewDatabaseLifecycle(cfg, log)

	ctx := context.Background()
	err := db.Start(ctx)
	assert.Error(t, err, "Expected database start to fail with invalid config")

	checker := NewDatabaseChecker(db, "test-db")
	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, "database connection is not initialized", result.Message)
}

func TestNewAPIChecker(t *testing.T) {
	url := "https://example.com"
	checker := NewAPIChecker(url, "test-api")

	require.NotNil(t, checker)
	assert.Equal(t, "test-api", checker.Name())
	assert.Equal(t, url, checker.endpoint)
}

func TestAPIChecker_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	checker := NewAPIChecker(server.URL, "test-api")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusHealthy, result.Status)
	assert.Contains(t, result.Message, "api responding with status")
	assert.Empty(t, result.Error)
}

func TestAPIChecker_Check_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	checker := NewAPIChecker(server.URL, "test-api")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "api returned status")
	assert.Empty(t, result.Error)
}

func TestAPIChecker_Check_InvalidURL(t *testing.T) {
	checker := NewAPIChecker("http://invalid-host-that-does-not-exist.local", "test-api")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, "api request failed", result.Message)
	assert.NotEmpty(t, result.Error)
}

func TestAPIChecker_Check_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewAPIChecker(server.URL, "test-api")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, "api request failed", result.Message)
	assert.Contains(t, result.Error, "context deadline exceeded")
}

func TestNewMemoryChecker(t *testing.T) {
	checker := NewMemoryChecker()

	require.NotNil(t, checker)
	assert.Equal(t, "memory_storage", checker.Name())
}

func TestMemoryChecker_Check_Success(t *testing.T) {
	checker := NewMemoryChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusHealthy, result.Status)
	assert.Equal(t, "memory storage operational", result.Message)
	assert.Empty(t, result.Error)
}

func TestMemoryChecker_Check_ContextCancelled(t *testing.T) {
	checker := NewMemoryChecker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.Check(ctx)

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, "memory storage check cancelled", result.Message)
}

func TestHealthCheckers_InterfaceCompliance(t *testing.T) {
	cfg := &config.DatabaseConfig{}
	log := logger.NewNop()
	db := database.NewDatabaseLifecycle(cfg, log)

	var checkers []health.Checker

	checkers = append(checkers, NewDatabaseChecker(db, "db"))
	checkers = append(checkers, NewAPIChecker("https://example.com", "api"))
	checkers = append(checkers, NewMemoryChecker())

	for i, checker := range checkers {
		assert.NotEmpty(t, checker.Name(), "Checker %d should have a name", i)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		result := checker.Check(ctx)
		cancel()

		assert.True(t, result.Status == health.StatusHealthy || result.Status == health.StatusUnhealthy,
			"Checker %d should return valid status", i)
	}
}

func BenchmarkDatabaseChecker_Check(b *testing.B) {
	cfg := &config.DatabaseConfig{}
	log := logger.NewNop()
	db := database.NewDatabaseLifecycle(cfg, log)
	checker := NewDatabaseChecker(db, "bench-db")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkAPIChecker_Check(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewAPIChecker(server.URL, "bench-api")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkMemoryChecker_Check(b *testing.B) {
	checker := NewMemoryChecker()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

type DatabaseCheckerTestSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	dbLifecycle *database.Lifecycle
}

func (s *DatabaseCheckerTestSuite) SetupSuite() {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	host, err := pgContainer.Host(ctx)
	s.Require().NoError(err)
	port, err := pgContainer.MappedPort(ctx, "5432")
	s.Require().NoError(err)

	dbConfig := &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		},
	}

	log := logger.NewNop()
	s.dbLifecycle = database.NewDatabaseLifecycle(dbConfig, log)
	err = s.dbLifecycle.Start(ctx)
	s.Require().NoError(err)
}

func (s *DatabaseCheckerTestSuite) TearDownSuite() {
	ctx := context.Background()
	err := s.dbLifecycle.Stop(ctx)
	s.Require().NoError(err)
	err = s.pgContainer.Terminate(ctx)
	s.Require().NoError(err)
}

func TestDatabaseCheckerTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(DatabaseCheckerTestSuite))
}

func (s *DatabaseCheckerTestSuite) TestDatabaseChecker_Check_Success() {
	checker := NewDatabaseChecker(s.dbLifecycle, "test-db-integration")
	ctx := context.Background()

	result := checker.Check(ctx)

	s.Assert().Equal(health.StatusHealthy, result.Status)
	s.Assert().Equal("database connection healthy", result.Message)
	s.Assert().Empty(result.Error)
}
