package config

import (
	"microservice/internal/platform/logger"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DatabaseConfigTestSuite struct {
	suite.Suite
	originalEnv map[string]string
}

func (s *DatabaseConfigTestSuite) SetupTest() {
	s.originalEnv = make(map[string]string)
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT", "USER",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD",
		"POSTGRES_DB", "POSTGRES_SSL_MODE", "POSTGRES_MAX_OPEN_CONNS",
		"POSTGRES_MAX_IDLE_CONNS", "POSTGRES_CONN_MAX_LIFETIME", "POSTGRES_CONN_MAX_IDLE_TIME",
	}

	for _, env := range envVars {
		if val, exists := os.LookupEnv(env); exists {
			s.originalEnv[env] = val
		}
		s.Require().NoError(os.Unsetenv(env))
	}
}

func (s *DatabaseConfigTestSuite) TearDownTest() {
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT", "USER",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD",
		"POSTGRES_DB", "POSTGRES_SSL_MODE", "POSTGRES_MAX_OPEN_CONNS",
		"POSTGRES_MAX_IDLE_CONNS", "POSTGRES_CONN_MAX_LIFETIME", "POSTGRES_CONN_MAX_IDLE_TIME",
	}

	for _, env := range envVars {
		s.Require().NoError(os.Unsetenv(env))
	}

	for env, val := range s.originalEnv {
		s.Require().NoError(os.Setenv(env, val))
	}
}

func (s *DatabaseConfigTestSuite) TestLoadDatabase_DefaultValues() {
	cfg, err := LoadDatabase()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().Equal(EnvDevelopment, cfg.Environment)
	s.Assert().Equal(logger.LevelInfo, cfg.Logger.Level)
	s.Assert().Equal(logger.FormatJSON, cfg.Logger.Format)

	s.Assert().Equal("localhost", cfg.Postgres.Host)
	s.Assert().Equal(5432, cfg.Postgres.Port)
	s.Assert().Equal("postgres", cfg.Postgres.User)
	s.Assert().Equal("", cfg.Postgres.Password)
	s.Assert().Equal("microservice", cfg.Postgres.Database)
	s.Assert().Equal("disable", cfg.Postgres.SSLMode)
	s.Assert().Equal(25, cfg.Postgres.MaxOpenConns)
	s.Assert().Equal(5, cfg.Postgres.MaxIdleConns)
	s.Assert().Equal(5*time.Minute, cfg.Postgres.ConnMaxLifetime)
	s.Assert().Equal(5*time.Minute, cfg.Postgres.ConnMaxIdleTime)
}

func (s *DatabaseConfigTestSuite) TestLoadDatabase_WithEnvironmentVariables() {
	envVars := map[string]string{
		"ENV":                         EnvProduction,
		"LOGGER_LEVEL":                "error",
		"LOGGER_FORMAT":               "text",
		"POSTGRES_HOST":               "db.example.com",
		"POSTGRES_PORT":               "5433",
		"POSTGRES_USER":               "myuser",
		"POSTGRES_PASSWORD":           "mypassword",
		"POSTGRES_DB":                 "mydatabase",
		"POSTGRES_SSL_MODE":           "require",
		"POSTGRES_MAX_OPEN_CONNS":     "50",
		"POSTGRES_MAX_IDLE_CONNS":     "10",
		"POSTGRES_CONN_MAX_LIFETIME":  "10m",
		"POSTGRES_CONN_MAX_IDLE_TIME": "15m",
	}

	for key, value := range envVars {
		s.Require().NoError(os.Setenv(key, value))
	}

	cfg, err := LoadDatabase()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().Equal(EnvProduction, cfg.Environment)
	s.Assert().Equal(logger.LevelError, cfg.Logger.Level)
	s.Assert().Equal(logger.FormatText, cfg.Logger.Format)

	s.Assert().Equal("db.example.com", cfg.Postgres.Host)
	s.Assert().Equal(5433, cfg.Postgres.Port)
	s.Assert().Equal("myuser", cfg.Postgres.User)
	s.Assert().Equal("mypassword", cfg.Postgres.Password)
	s.Assert().Equal("mydatabase", cfg.Postgres.Database)
	s.Assert().Equal("require", cfg.Postgres.SSLMode)
	s.Assert().Equal(50, cfg.Postgres.MaxOpenConns)
	s.Assert().Equal(10, cfg.Postgres.MaxIdleConns)
	s.Assert().Equal(10*time.Minute, cfg.Postgres.ConnMaxLifetime)
	s.Assert().Equal(15*time.Minute, cfg.Postgres.ConnMaxIdleTime)

	for key := range envVars {
		s.Require().NoError(os.Unsetenv(key))
	}
}

func (s *DatabaseConfigTestSuite) TestPostgresConfig_DSN() {
	tests := []struct {
		name     string
		config   PostgresConfig
		expected string
	}{
		{
			name: "default_values",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "",
				Database: "microservice",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password= dbname=microservice sslmode=disable",
		},
		{
			name: "with_password",
			config: PostgresConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "myuser",
				Password: "mypassword",
				Database: "mydatabase",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=myuser password=mypassword dbname=mydatabase sslmode=require",
		},
		{
			name: "special_characters_in_values",
			config: PostgresConfig{
				Host:     "db-host.example.com",
				Port:     5432,
				User:     "user_name",
				Password: "pass@word!123",
				Database: "my-database",
				SSLMode:  "verify-full",
			},
			expected: "host=db-host.example.com port=5432 user=user_name password=pass@word!123 dbname=my-database sslmode=verify-full",
		},
		{
			name: "zero_port",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     0,
				User:     "postgres",
				Password: "",
				Database: "test",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=0 user=postgres password= dbname=test sslmode=disable",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dsn := tt.config.DSN()
			s.Assert().Equal(tt.expected, dsn)
		})
	}
}

func (s *DatabaseConfigTestSuite) TestPostgresConfig_Getters() {
	config := PostgresConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 15 * time.Minute,
	}

	s.Assert().Equal(25, config.GetMaxOpenConns())
	s.Assert().Equal(5, config.GetMaxIdleConns())
	s.Assert().Equal(10*time.Minute, config.GetConnMaxLifetime())
	s.Assert().Equal(15*time.Minute, config.GetConnMaxIdleTime())
}

func (s *DatabaseConfigTestSuite) TestPostgresConfig_EdgeCases() {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*DatabaseConfig)
	}{
		{
			name: "zero_connections",
			envVars: map[string]string{
				"POSTGRES_MAX_OPEN_CONNS": "0",
				"POSTGRES_MAX_IDLE_CONNS": "0",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal(0, cfg.Postgres.MaxOpenConns)
				s.Assert().Equal(0, cfg.Postgres.MaxIdleConns)
				s.Assert().Equal(0, cfg.Postgres.GetMaxOpenConns())
				s.Assert().Equal(0, cfg.Postgres.GetMaxIdleConns())
			},
		},
		{
			name: "high_connections",
			envVars: map[string]string{
				"POSTGRES_MAX_OPEN_CONNS": "1000",
				"POSTGRES_MAX_IDLE_CONNS": "100",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal(1000, cfg.Postgres.MaxOpenConns)
				s.Assert().Equal(100, cfg.Postgres.MaxIdleConns)
			},
		},
		{
			name: "zero_timeouts",
			envVars: map[string]string{
				"POSTGRES_CONN_MAX_LIFETIME":  "0s",
				"POSTGRES_CONN_MAX_IDLE_TIME": "0s",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal(0*time.Second, cfg.Postgres.ConnMaxLifetime)
				s.Assert().Equal(0*time.Second, cfg.Postgres.ConnMaxIdleTime)
			},
		},
		{
			name: "large_timeouts",
			envVars: map[string]string{
				"POSTGRES_CONN_MAX_LIFETIME":  "1h",
				"POSTGRES_CONN_MAX_IDLE_TIME": "30m",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal(1*time.Hour, cfg.Postgres.ConnMaxLifetime)
				s.Assert().Equal(30*time.Minute, cfg.Postgres.ConnMaxIdleTime)
			},
		},
		{
			name: "different_ssl_modes",
			envVars: map[string]string{
				"POSTGRES_SSL_MODE": "verify-ca",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal("verify-ca", cfg.Postgres.SSLMode)
			},
		},
		{
			name: "empty_password",
			envVars: map[string]string{
				"POSTGRES_PASSWORD": "",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal("", cfg.Postgres.Password)
				dsn := cfg.Postgres.DSN()
				s.Assert().Contains(dsn, "password=")
			},
		},
		{
			name: "non_standard_port",
			envVars: map[string]string{
				"POSTGRES_PORT": "65535",
			},
			check: func(cfg *DatabaseConfig) {
				s.Assert().Equal(65535, cfg.Postgres.Port)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for key, value := range tt.envVars {
				s.Require().NoError(os.Setenv(key, value))
			}

			cfg, err := LoadDatabase()
			s.Require().NoError(err)
			s.Require().NotNil(cfg)

			tt.check(cfg)

			for key := range tt.envVars {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

func (s *DatabaseConfigTestSuite) TestDatabaseConfig_InheritsBaseConfig() {
	s.Require().NoError(os.Setenv("ENV", EnvTest))
	defer func() { s.Require().NoError(os.Unsetenv("ENV")) }()

	cfg, err := LoadDatabase()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().True(cfg.IsTest())
	s.Assert().False(cfg.IsProduction())
	s.Assert().False(cfg.IsDevelopment())
	s.Assert().False(cfg.IsStaging())
}

func (s *DatabaseConfigTestSuite) TestZeroValues() {
	cfg := DatabaseConfig{}

	s.Assert().Equal("", cfg.Environment)
	s.Assert().Equal("", cfg.Postgres.Host)
	s.Assert().Equal(0, cfg.Postgres.Port)
	s.Assert().Equal("", cfg.Postgres.User)
	s.Assert().Equal("", cfg.Postgres.Password)
	s.Assert().Equal("", cfg.Postgres.Database)
	s.Assert().Equal("", cfg.Postgres.SSLMode)
	s.Assert().Equal(0, cfg.Postgres.MaxOpenConns)
	s.Assert().Equal(0, cfg.Postgres.MaxIdleConns)
	s.Assert().Equal(time.Duration(0), cfg.Postgres.ConnMaxLifetime)
	s.Assert().Equal(time.Duration(0), cfg.Postgres.ConnMaxIdleTime)
}

func (s *DatabaseConfigTestSuite) TestDSN_SpecialCharacters() {
	config := PostgresConfig{
		Host:     "host with spaces",
		Port:     5432,
		User:     "user@domain.com",
		Password: "password with spaces & symbols!",
		Database: "database-name_123",
		SSLMode:  "require",
	}

	dsn := config.DSN()
	expectedDSN := "host=host with spaces port=5432 user=user@domain.com password=password with spaces & symbols! dbname=database-name_123 sslmode=require"
	s.Assert().Equal(expectedDSN, dsn)
}

func (s *DatabaseConfigTestSuite) TestDSN_Components() {
	config := PostgresConfig{
		Host:     "example.com",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	dsn := config.DSN()

	s.Assert().Contains(dsn, "host=example.com")
	s.Assert().Contains(dsn, "port=5432")
	s.Assert().Contains(dsn, "user=testuser")
	s.Assert().Contains(dsn, "password=testpass")
	s.Assert().Contains(dsn, "dbname=testdb")
	s.Assert().Contains(dsn, "sslmode=disable")

	parts := strings.Split(dsn, " ")
	s.Assert().Len(parts, 6)
	s.Assert().Equal("host=example.com", parts[0])
	s.Assert().Equal("port=5432", parts[1])
	s.Assert().Equal("user=testuser", parts[2])
	s.Assert().Equal("password=testpass", parts[3])
	s.Assert().Equal("dbname=testdb", parts[4])
	s.Assert().Equal("sslmode=disable", parts[5])
}

func (s *DatabaseConfigTestSuite) TestTimeoutParsing() {
	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"1s", 1 * time.Second},
		{"30s", 30 * time.Second},
		{"1m", 1 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"90m", 90 * time.Minute},
		{"2h30m", 2*time.Hour + 30*time.Minute},
	}

	for _, tc := range testCases {
		s.Run("timeout_"+tc.input, func() {
			s.Require().NoError(os.Setenv("POSTGRES_CONN_MAX_LIFETIME", tc.input))
			defer func() { s.Require().NoError(os.Unsetenv("POSTGRES_CONN_MAX_LIFETIME")) }()

			cfg, err := LoadDatabase()
			s.Require().NoError(err)
			s.Assert().Equal(tc.expected, cfg.Postgres.ConnMaxLifetime)
		})
	}
}

func (s *DatabaseConfigTestSuite) TestLoadDatabase_Performance() {
	for i := 0; i < 100; i++ {
		cfg, err := LoadDatabase()
		s.Require().NoError(err)
		s.Require().NotNil(cfg)
	}
}

func (s *DatabaseConfigTestSuite) TestDSN_Performance() {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		Database: "testdb",
		SSLMode:  "disable",
	}

	for i := 0; i < 1000; i++ {
		dsn := config.DSN()
		s.Assert().NotEmpty(dsn)
	}
}

func BenchmarkLoadDatabase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := LoadDatabase()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresConfig_DSN(b *testing.B) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		Database: "testdb",
		SSLMode:  "disable",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.DSN()
	}
}

func BenchmarkPostgresConfig_Getters(b *testing.B) {
	config := PostgresConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetMaxOpenConns()
		_ = config.GetMaxIdleConns()
		_ = config.GetConnMaxLifetime()
		_ = config.GetConnMaxIdleTime()
	}
}

func TestDatabaseConfigTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseConfigTestSuite))
}
