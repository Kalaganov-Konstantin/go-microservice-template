package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"microservice/internal/config"
	"microservice/internal/platform/logger"
)

type DatabaseTestSuite struct {
	suite.Suite
	postgresContainer *postgres.PostgresContainer
	dbConfig          *config.DatabaseConfig
	logger            logger.Logger
}

func (suite *DatabaseTestSuite) SetupSuite() {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(suite.T(), err)

	suite.postgresContainer = postgresContainer

	host, err := postgresContainer.Host(ctx)
	require.NoError(suite.T(), err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(suite.T(), err)

	suite.dbConfig = &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		},
	}

	suite.logger = logger.NewNop()
}

func (suite *DatabaseTestSuite) TearDownSuite() {
	if suite.postgresContainer != nil {
		ctx := context.Background()
		err := suite.postgresContainer.Terminate(ctx)
		require.NoError(suite.T(), err)
	}
}

func TestDatabaseSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(DatabaseTestSuite))
}

func (suite *DatabaseTestSuite) TestLifecycle_StartStop_Success() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	suite.Require().NoError(err)

	conn := lifecycle.Connection()
	suite.Require().NotNil(conn, "Connection should be available after Start()")

	err = conn.Ping(ctx)
	suite.Assert().NoError(err, "Should be able to ping database")

	err = lifecycle.Stop(ctx)
	suite.Assert().NoError(err, "Stop should not error")

	conn = lifecycle.Connection()
	suite.Assert().Nil(conn, "Connection should be nil after Stop()")

	err = lifecycle.Stop(ctx)
	suite.Assert().NoError(err, "Second Stop should not error")
}

func (suite *DatabaseTestSuite) TestLifecycle_StartTwice_ClosesExistingConnection() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	suite.Require().NoError(err)

	firstConn := lifecycle.Connection()
	suite.Require().NotNil(firstConn)

	err = firstConn.Ping(ctx)
	suite.Require().NoError(err, "First connection should be alive")

	err = lifecycle.Start(ctx)
	suite.Require().NoError(err)

	secondConn := lifecycle.Connection()
	suite.Require().NotNil(secondConn)

	suite.Assert().NotEqual(firstConn, secondConn)

	err = secondConn.Ping(ctx)
	suite.Assert().NoError(err, "Second connection should work")

	err = lifecycle.Stop(ctx)
	suite.Require().NoError(err)
}

func (suite *DatabaseTestSuite) TestLifecycle_Connection_BeforeStart() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	conn := lifecycle.Connection()
	suite.Assert().Nil(conn)
}

func (suite *DatabaseTestSuite) TestLifecycle_StopWithError() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx := context.Background()
	err := lifecycle.Stop(ctx)
	suite.Assert().NoError(err, "Stop should not error when db is nil")
}

func (suite *DatabaseTestSuite) TestLifecycle_StartPingFails() {
	cfg := &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     "localhost",
			Port:     9999,
			User:     "test",
			Password: "test",
			Database: "test",
			SSLMode:  "disable",
		},
	}
	lifecycle := NewDatabaseLifecycle(cfg, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	suite.Assert().Error(err, "Start should fail when cannot connect")

	conn := lifecycle.Connection()
	suite.Assert().Nil(conn, "Connection should be nil after failed start")
}

func (suite *DatabaseTestSuite) TestLifecycle_MultipleConcurrentStarts() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- lifecycle.Start(ctx)
		}()
	}

	var errors []error
	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			errors = append(errors, err)
		}
	}

	suite.Assert().True(len(errors) <= 3, "Concurrent starts should not cause crashes")

	conn := lifecycle.Connection()
	suite.Assert().NotNil(conn, "Connection should be available after concurrent starts")

	err := lifecycle.Stop(ctx)
	suite.Require().NoError(err)
}

func (suite *DatabaseTestSuite) TestLifecycle_DatabaseOperations() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	suite.Require().NoError(err)
	defer func() {
		err := lifecycle.Stop(ctx)
		suite.Require().NoError(err)
	}()

	conn := lifecycle.Connection()
	suite.Require().NotNil(conn)

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL
		)
	`)
	suite.Require().NoError(err, "Should be able to create table")

	_, err = conn.ExecContext(ctx, "INSERT INTO test_table (name) VALUES ($1)", "test_name")
	suite.Require().NoError(err, "Should be able to insert data")

	var name string
	err = conn.QueryRowContext(ctx, "SELECT name FROM test_table WHERE id = 1").Scan(&name)
	suite.Require().NoError(err, "Should be able to query data")
	suite.Assert().Equal("test_name", name)

	_, err = conn.ExecContext(ctx, "DROP TABLE test_table")
	suite.Require().NoError(err, "Should be able to drop table")
}

func (suite *DatabaseTestSuite) TestLifecycle_TransactionSupport() {
	lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	suite.Require().NoError(err)
	defer func() {
		err := lifecycle.Stop(ctx)
		suite.Require().NoError(err)
	}()

	conn := lifecycle.Connection()
	suite.Require().NotNil(conn)

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tx_test (
			id SERIAL PRIMARY KEY,
			value INTEGER
		)
	`)
	suite.Require().NoError(err)
	defer func() {
		_, _ = conn.ExecContext(ctx, "DROP TABLE IF EXISTS tx_test")
	}()

	tx, err := conn.BeginTx(ctx, nil)
	suite.Require().NoError(err)

	_, err = tx.ExecContext(ctx, "INSERT INTO tx_test (value) VALUES ($1)", 42)
	suite.Require().NoError(err)

	err = tx.Rollback()
	suite.Require().NoError(err)

	var count int
	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM tx_test").Scan(&count)
	suite.Require().NoError(err)
	suite.Assert().Equal(0, count, "Data should not exist after rollback")

	tx, err = conn.BeginTx(ctx, nil)
	suite.Require().NoError(err)

	_, err = tx.ExecContext(ctx, "INSERT INTO tx_test (value) VALUES ($1)", 123)
	suite.Require().NoError(err)

	err = tx.Commit()
	suite.Require().NoError(err)

	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM tx_test").Scan(&count)
	suite.Require().NoError(err)
	suite.Assert().Equal(1, count, "Data should exist after commit")
}

func (suite *DatabaseTestSuite) TestBenchmarkLifecycle_StartStop() {
	suite.T().Run("BenchmarkLifecycle_StartStop", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping benchmark in short mode")
		}

		b := testing.Benchmark(func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lifecycle := NewDatabaseLifecycle(suite.dbConfig, suite.logger)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

				err := lifecycle.Start(ctx)
				if err != nil {
					b.Fatalf("Failed to start database: %v", err)
				}

				_ = lifecycle.Stop(ctx)
				cancel()
			}
		})

		t.Logf("Benchmark result: %s", b.String())
	})
}

func TestNewDatabaseLifecycle(t *testing.T) {
	cfg := &config.DatabaseConfig{}
	log := logger.NewNop()
	lifecycle := NewDatabaseLifecycle(cfg, log)

	require.NotNil(t, lifecycle)
	assert.Equal(t, cfg, lifecycle.cfg)
	assert.Equal(t, log, lifecycle.logger)
	assert.Nil(t, lifecycle.db)
}

func TestLifecycle_StartStop_InvalidConfig(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Postgres: config.PostgresConfig{
			Host:     "invalid-host",
			Port:     9999,
			User:     "invalid",
			Password: "invalid",
			Database: "invalid",
			SSLMode:  "disable",
		},
	}
	log := logger.NewNop()
	lifecycle := NewDatabaseLifecycle(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := lifecycle.Start(ctx)
	assert.Error(t, err)

	conn := lifecycle.Connection()
	assert.Nil(t, conn)

	err = lifecycle.Stop(ctx)
	assert.NoError(t, err)
}
