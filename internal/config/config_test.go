package config

import (
	"microservice/internal/platform/logger"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	originalEnv map[string]string
}

func (s *ConfigTestSuite) SetupTest() {
	s.originalEnv = make(map[string]string)
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT",
	}

	for _, env := range envVars {
		if val, exists := os.LookupEnv(env); exists {
			s.originalEnv[env] = val
		}
		s.Require().NoError(os.Unsetenv(env))
	}
}

func (s *ConfigTestSuite) TearDownTest() {
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT",
	}

	for _, env := range envVars {
		s.Require().NoError(os.Unsetenv(env))
	}

	for env, val := range s.originalEnv {
		s.Require().NoError(os.Setenv(env, val))
	}
}

func (s *ConfigTestSuite) TestLoadBase_DefaultValues() {
	cfg, err := LoadBase()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)
	s.Assert().Equal(EnvDevelopment, cfg.Environment)
	s.Assert().Equal(logger.LevelInfo, cfg.Logger.Level)
	s.Assert().Equal(logger.FormatJSON, cfg.Logger.Format)
}

func (s *ConfigTestSuite) TestLoadBase_WithEnvironmentVariables() {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected BaseConfig
	}{
		{
			name: "production_environment",
			envVars: map[string]string{
				"ENV":           EnvProduction,
				"LOGGER_LEVEL":  "error",
				"LOGGER_FORMAT": "text",
			},
			expected: BaseConfig{
				Environment: EnvProduction,
				Logger: LoggerConfig{
					Level:  logger.LevelError,
					Format: logger.FormatText,
				},
			},
		},
		{
			name: "staging_environment",
			envVars: map[string]string{
				"ENV":           EnvStaging,
				"LOGGER_LEVEL":  "debug",
				"LOGGER_FORMAT": "json",
			},
			expected: BaseConfig{
				Environment: EnvStaging,
				Logger: LoggerConfig{
					Level:  logger.LevelDebug,
					Format: logger.FormatJSON,
				},
			},
		},
		{
			name: "test_environment",
			envVars: map[string]string{
				"ENV":           EnvTest,
				"LOGGER_LEVEL":  "warn",
				"LOGGER_FORMAT": "text",
			},
			expected: BaseConfig{
				Environment: EnvTest,
				Logger: LoggerConfig{
					Level:  logger.LevelWarn,
					Format: logger.FormatText,
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for key, value := range tt.envVars {
				s.Require().NoError(os.Setenv(key, value))
			}

			cfg, err := LoadBase()

			s.Require().NoError(err)
			s.Require().NotNil(cfg)
			s.Assert().Equal(tt.expected.Environment, cfg.Environment)
			s.Assert().Equal(tt.expected.Logger.Level, cfg.Logger.Level)
			s.Assert().Equal(tt.expected.Logger.Format, cfg.Logger.Format)

			for key := range tt.envVars {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

func (s *ConfigTestSuite) TestEnvironmentCheckers() {
	tests := []struct {
		name        string
		environment string
		checkers    map[string]bool
	}{
		{
			name:        "development_environment",
			environment: EnvDevelopment,
			checkers: map[string]bool{
				"IsDevelopment": true,
				"IsProduction":  false,
				"IsStaging":     false,
				"IsTest":        false,
			},
		},
		{
			name:        "production_environment",
			environment: EnvProduction,
			checkers: map[string]bool{
				"IsDevelopment": false,
				"IsProduction":  true,
				"IsStaging":     false,
				"IsTest":        false,
			},
		},
		{
			name:        "staging_environment",
			environment: EnvStaging,
			checkers: map[string]bool{
				"IsDevelopment": false,
				"IsProduction":  false,
				"IsStaging":     true,
				"IsTest":        false,
			},
		},
		{
			name:        "test_environment",
			environment: EnvTest,
			checkers: map[string]bool{
				"IsDevelopment": false,
				"IsProduction":  false,
				"IsStaging":     false,
				"IsTest":        true,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := &BaseConfig{Environment: tt.environment}

			s.Assert().Equal(tt.checkers["IsDevelopment"], cfg.IsDevelopment())
			s.Assert().Equal(tt.checkers["IsProduction"], cfg.IsProduction())
			s.Assert().Equal(tt.checkers["IsStaging"], cfg.IsStaging())
			s.Assert().Equal(tt.checkers["IsTest"], cfg.IsTest())
		})
	}
}

func (s *ConfigTestSuite) TestEnvironmentCheckers_CaseInsensitive() {
	tests := []struct {
		name        string
		environment string
		expected    string
	}{
		{"uppercase_development", "DEVELOPMENT", "IsDevelopment"},
		{"mixed_case_production", "Production", "IsProduction"},
		{"uppercase_staging", "STAGING", "IsStaging"},
		{"mixed_case_test", "Test", "IsTest"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := &BaseConfig{Environment: tt.environment}

			switch tt.expected {
			case "IsDevelopment":
				s.Assert().True(cfg.IsDevelopment())
				s.Assert().False(cfg.IsProduction())
				s.Assert().False(cfg.IsStaging())
				s.Assert().False(cfg.IsTest())
			case "IsProduction":
				s.Assert().False(cfg.IsDevelopment())
				s.Assert().True(cfg.IsProduction())
				s.Assert().False(cfg.IsStaging())
				s.Assert().False(cfg.IsTest())
			case "IsStaging":
				s.Assert().False(cfg.IsDevelopment())
				s.Assert().False(cfg.IsProduction())
				s.Assert().True(cfg.IsStaging())
				s.Assert().False(cfg.IsTest())
			case "IsTest":
				s.Assert().False(cfg.IsDevelopment())
				s.Assert().False(cfg.IsProduction())
				s.Assert().False(cfg.IsStaging())
				s.Assert().True(cfg.IsTest())
			}
		})
	}
}

func (s *ConfigTestSuite) TestConstants() {
	s.Assert().Equal("development", EnvDevelopment)
	s.Assert().Equal("staging", EnvStaging)
	s.Assert().Equal("production", EnvProduction)
	s.Assert().Equal("test", EnvTest)
}

func (s *ConfigTestSuite) TestLoggerConfig_DefaultValues() {
	cfg := LoggerConfig{}
	s.Assert().Equal(logger.Level(""), cfg.Level)
	s.Assert().Equal(logger.Format(""), cfg.Format)
}

func (s *ConfigTestSuite) TestBaseConfig_ZeroValues() {
	cfg := BaseConfig{}
	s.Assert().Equal("", cfg.Environment)
	s.Assert().False(cfg.IsDevelopment())
	s.Assert().False(cfg.IsProduction())
	s.Assert().False(cfg.IsStaging())
	s.Assert().False(cfg.IsTest())
}

func (s *ConfigTestSuite) TestEnvironmentCheckers_UnknownEnvironment() {
	cfg := &BaseConfig{Environment: "unknown"}

	s.Assert().False(cfg.IsDevelopment())
	s.Assert().False(cfg.IsProduction())
	s.Assert().False(cfg.IsStaging())
	s.Assert().False(cfg.IsTest())
}

func (s *ConfigTestSuite) TestEnvironmentCheckers_Performance() {
	cfg := &BaseConfig{Environment: EnvProduction}

	for i := 0; i < 1000; i++ {
		cfg.IsProduction()
		cfg.IsDevelopment()
		cfg.IsStaging()
		cfg.IsTest()
	}
}

func BenchmarkLoadBase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := LoadBase()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEnvironmentCheckers(b *testing.B) {
	cfg := &BaseConfig{Environment: EnvProduction}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.IsProduction()
		cfg.IsDevelopment()
		cfg.IsStaging()
		cfg.IsTest()
	}
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
