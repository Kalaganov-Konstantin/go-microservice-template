package postgres

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

type MockConfig struct {
	dsn             string
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

func (m *MockConfig) DSN() string {
	return m.dsn
}

func (m *MockConfig) GetMaxOpenConns() int {
	return m.maxOpenConns
}

func (m *MockConfig) GetMaxIdleConns() int {
	return m.maxIdleConns
}

func (m *MockConfig) GetConnMaxLifetime() time.Duration {
	return m.connMaxLifetime
}

func (m *MockConfig) GetConnMaxIdleTime() time.Duration {
	return m.connMaxIdleTime
}

type PostgresConnectionTestSuite struct {
	suite.Suite
	mockConfig *MockConfig
}

func (s *PostgresConnectionTestSuite) SetupTest() {
	s.mockConfig = &MockConfig{
		dsn:             "postgres://user:password@localhost:5432/testdb?sslmode=disable",
		maxOpenConns:    25,
		maxIdleConns:    5,
		connMaxLifetime: 5 * time.Minute,
		connMaxIdleTime: 5 * time.Minute,
	}
}

func (s *PostgresConnectionTestSuite) TestNew_Success() {
	cfg := &MockConfig{
		dsn:             "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		maxOpenConns:    50,
		maxIdleConns:    10,
		connMaxLifetime: 10 * time.Minute,
		connMaxIdleTime: 2 * time.Minute,
	}

	db, err := New(cfg)

	s.Assert().NoError(err)
	s.Assert().NotNil(db)
	s.Assert().NotNil(db.DB)
	s.Assert().Equal(cfg, db.config)

	s.Assert().NotPanics(func() {
		stats := db.DB.Stats()
		s.Assert().GreaterOrEqual(stats.MaxOpenConnections, 0)
	})

	s.Require().NoError(db.Close())
}

func (s *PostgresConnectionTestSuite) TestNew_ZeroValues() {
	cfg := &MockConfig{
		dsn:             "postgres://localhost/test",
		maxOpenConns:    0,
		maxIdleConns:    0,
		connMaxLifetime: 0,
		connMaxIdleTime: 0,
	}

	db, err := New(cfg)
	s.Assert().NoError(err)
	s.Assert().NotNil(db)
	s.Assert().Equal(cfg, db.config)

	s.Require().NoError(db.Close())
}

func (s *PostgresConnectionTestSuite) TestNew_HighValues() {
	cfg := &MockConfig{
		dsn:             "postgres://localhost/test",
		maxOpenConns:    1000,
		maxIdleConns:    100,
		connMaxLifetime: 24 * time.Hour,
		connMaxIdleTime: 1 * time.Hour,
	}

	db, err := New(cfg)
	s.Assert().NoError(err)
	s.Assert().NotNil(db)
	s.Assert().Equal(cfg, db.config)

	s.Require().NoError(db.Close())
}

func (s *PostgresConnectionTestSuite) TestPing_Success() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx := context.Background()

	mock.ExpectPing()
	err = db.Ping(ctx)
	s.Assert().NoError(err)

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func (s *PostgresConnectionTestSuite) TestPing_DatabaseError() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx := context.Background()

	expectedError := driver.ErrBadConn
	mock.ExpectPing().WillReturnError(expectedError)

	err = db.Ping(ctx)
	s.Assert().Error(err)
	s.Assert().Equal(expectedError, err)

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func (s *PostgresConnectionTestSuite) TestPing_WithTimeout() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	mock.ExpectPing().WillDelayFor(10 * time.Millisecond)

	err = db.Ping(ctx)
	s.Assert().Error(err)
	s.Assert().True(
		strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "canceling query due to user request") ||
			strings.Contains(err.Error(), "context canceled"),
		"Expected timeout error, got: %v", err,
	)
}

func (s *PostgresConnectionTestSuite) TestPing_CancelledContext() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock.ExpectPing()

	err = db.Ping(ctx)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "context canceled")
}

func (s *PostgresConnectionTestSuite) TestPing_InternalTimeout() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx := context.Background()

	mock.ExpectPing().WillDelayFor(6 * time.Second)

	start := time.Now()
	err = db.Ping(ctx)
	duration := time.Since(start)

	s.Assert().Error(err)
	s.Assert().Less(duration, 6*time.Second, "Ping should timeout before 6 seconds")
	s.Assert().Greater(duration, 4*time.Second, "Ping should take at least ~5 seconds to timeout")
}

func (s *PostgresConnectionTestSuite) TestClose_Success() {
	mockDB, mock, err := sqlmock.New()
	s.Require().NoError(err)

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	mock.ExpectClose()
	err = db.Close()
	s.Assert().NoError(err)

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func (s *PostgresConnectionTestSuite) TestClose_Error() {
	mockDB, mock, err := sqlmock.New()
	s.Require().NoError(err)

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	expectedError := driver.ErrBadConn
	mock.ExpectClose().WillReturnError(expectedError)

	err = db.Close()
	s.Assert().Error(err)
	s.Assert().Equal(expectedError, err)

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func (s *PostgresConnectionTestSuite) TestConfig_Interface() {
	var cfg Config = s.mockConfig

	s.Assert().Equal(s.mockConfig.dsn, cfg.DSN())
	s.Assert().Equal(s.mockConfig.maxOpenConns, cfg.GetMaxOpenConns())
	s.Assert().Equal(s.mockConfig.maxIdleConns, cfg.GetMaxIdleConns())
	s.Assert().Equal(s.mockConfig.connMaxLifetime, cfg.GetConnMaxLifetime())
	s.Assert().Equal(s.mockConfig.connMaxIdleTime, cfg.GetConnMaxIdleTime())
}

func (s *PostgresConnectionTestSuite) TestDB_StructFields() {
	mockDB, _, err := sqlmock.New()
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	s.Assert().NotNil(db.DB)
	s.Assert().Equal(mockDB, db.DB)
	s.Assert().Equal(s.mockConfig, db.config)
}

func (s *PostgresConnectionTestSuite) TestNew_EmptyDSN() {
	cfg := &MockConfig{
		dsn:             "",
		maxOpenConns:    10,
		maxIdleConns:    2,
		connMaxLifetime: 1 * time.Minute,
		connMaxIdleTime: 30 * time.Second,
	}

	db, err := New(cfg)

	if err != nil {
		s.Assert().Error(err)
		s.Assert().Nil(db)
	} else {
		s.Assert().NotNil(db)
		s.Require().NoError(db.Close())
	}
}

func (s *PostgresConnectionTestSuite) TestNew_InvalidDSN() {
	cfg := &MockConfig{
		dsn:             "invalid-dsn-format",
		maxOpenConns:    10,
		maxIdleConns:    2,
		connMaxLifetime: 1 * time.Minute,
		connMaxIdleTime: 30 * time.Second,
	}

	db, err := New(cfg)

	if err != nil {
		s.Assert().Error(err)
		s.Assert().Nil(db)
	} else {
		s.Assert().NotNil(db)
		s.Require().NoError(db.Close())
	}
}

func (s *PostgresConnectionTestSuite) TestPing_Performance() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		mock.ExpectPing()
	}

	for i := 0; i < 10; i++ {
		err := db.Ping(ctx)
		s.Assert().NoError(err)
	}

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func (s *PostgresConnectionTestSuite) TestPing_Concurrent() {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	s.Require().NoError(err)
	defer func() { _ = mockDB.Close() }()

	db := &DB{
		DB:     mockDB,
		config: s.mockConfig,
	}

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		mock.ExpectPing()
	}

	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			done <- db.Ping(ctx)
		}()
	}
	for i := 0; i < 5; i++ {
		err := <-done
		s.Assert().NoError(err)
	}

	s.Assert().NoError(mock.ExpectationsWereMet())
}

func BenchmarkNew(b *testing.B) {
	cfg := &MockConfig{
		dsn:             "postgres://user:pass@localhost/test",
		maxOpenConns:    25,
		maxIdleConns:    5,
		connMaxLifetime: 5 * time.Minute,
		connMaxIdleTime: 5 * time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := New(cfg)
		if err != nil {
			b.Fatal(err)
		}
		_ = db.Close()
	}
}

func BenchmarkPing(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = mockDB.Close() }()

	cfg := &MockConfig{
		dsn:             "postgres://user:pass@localhost/test",
		maxOpenConns:    25,
		maxIdleConns:    5,
		connMaxLifetime: 5 * time.Minute,
		connMaxIdleTime: 5 * time.Minute,
	}

	db := &DB{
		DB:     mockDB,
		config: cfg,
	}

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		mock.ExpectPing()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.Ping(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClose(b *testing.B) {
	cfg := &MockConfig{
		dsn:             "postgres://user:pass@localhost/test",
		maxOpenConns:    25,
		maxIdleConns:    5,
		connMaxLifetime: 5 * time.Minute,
		connMaxIdleTime: 5 * time.Minute,
	}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}

	db := &DB{
		DB:     mockDB,
		config: cfg,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ExpectClose()
		err = db.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPostgresConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresConnectionTestSuite))
}
