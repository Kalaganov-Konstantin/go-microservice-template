package config

import (
	"microservice/internal/platform/logger"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HttpConfigTestSuite struct {
	suite.Suite
	originalEnv map[string]string
}

func (s *HttpConfigTestSuite) SetupTest() {
	s.originalEnv = make(map[string]string)
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT",
		"HTTP_SERVER_HOST", "HTTP_SERVER_PORT",
		"HTTP_SERVER_READ_TIMEOUT", "HTTP_SERVER_WRITE_TIMEOUT", "HTTP_SERVER_IDLE_TIMEOUT",
		"RATE_LIMIT_GLOBAL_REQUESTS", "RATE_LIMIT_GLOBAL_WINDOW",
		"RATE_LIMIT_REQUESTS_PER_IP", "RATE_LIMIT_WINDOW_SECONDS",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS",
		"CORS_EXPOSED_HEADERS", "CORS_ALLOW_CREDENTIALS", "CORS_MAX_AGE",
	}

	for _, env := range envVars {
		if val, exists := os.LookupEnv(env); exists {
			s.originalEnv[env] = val
		}
		s.Require().NoError(os.Unsetenv(env))
	}
}

func (s *HttpConfigTestSuite) TearDownTest() {
	envVars := []string{
		"ENV", "LOGGER_LEVEL", "LOGGER_FORMAT",
		"HTTP_SERVER_HOST", "HTTP_SERVER_PORT",
		"HTTP_SERVER_READ_TIMEOUT", "HTTP_SERVER_WRITE_TIMEOUT", "HTTP_SERVER_IDLE_TIMEOUT",
		"RATE_LIMIT_GLOBAL_REQUESTS", "RATE_LIMIT_GLOBAL_WINDOW",
		"RATE_LIMIT_REQUESTS_PER_IP", "RATE_LIMIT_WINDOW_SECONDS",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS",
		"CORS_EXPOSED_HEADERS", "CORS_ALLOW_CREDENTIALS", "CORS_MAX_AGE",
	}

	for _, env := range envVars {
		s.Require().NoError(os.Unsetenv(env))
	}

	for env, val := range s.originalEnv {
		s.Require().NoError(os.Setenv(env, val))
	}
}

func (s *HttpConfigTestSuite) TestLoadHttp_DefaultValues() {
	cfg, err := LoadHttp()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().Equal(EnvDevelopment, cfg.Environment)
	s.Assert().Equal(logger.LevelInfo, cfg.Logger.Level)
	s.Assert().Equal(logger.FormatJSON, cfg.Logger.Format)

	s.Assert().Equal("0.0.0.0", cfg.Server.Host)
	s.Assert().Equal(8080, cfg.Server.Port)
	s.Assert().Equal(30, cfg.Server.ReadTimeout)
	s.Assert().Equal(30, cfg.Server.WriteTimeout)
	s.Assert().Equal(120, cfg.Server.IdleTimeout)

	s.Assert().Equal(1000, cfg.RateLimit.GlobalRequests)
	s.Assert().Equal(60, cfg.RateLimit.GlobalWindow)
	s.Assert().Equal(100, cfg.RateLimit.RequestsPerIP)
	s.Assert().Equal(60, cfg.RateLimit.WindowSeconds)

	s.Assert().Equal([]string{"*"}, cfg.CORS.AllowedOrigins)
	s.Assert().Equal([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, cfg.CORS.AllowedMethods)
	s.Assert().Equal([]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, cfg.CORS.AllowedHeaders)
	s.Assert().Empty(cfg.CORS.ExposedHeaders)
	s.Assert().False(cfg.CORS.AllowCredentials)
	s.Assert().Equal(86400, cfg.CORS.MaxAge)
}

func (s *HttpConfigTestSuite) TestLoadHttp_WithEnvironmentVariables() {
	envVars := map[string]string{
		"ENV":                        EnvProduction,
		"LOGGER_LEVEL":               "error",
		"LOGGER_FORMAT":              "text",
		"HTTP_SERVER_HOST":           "127.0.0.1",
		"HTTP_SERVER_PORT":           "9090",
		"HTTP_SERVER_READ_TIMEOUT":   "60",
		"HTTP_SERVER_WRITE_TIMEOUT":  "60",
		"HTTP_SERVER_IDLE_TIMEOUT":   "300",
		"RATE_LIMIT_GLOBAL_REQUESTS": "2000",
		"RATE_LIMIT_GLOBAL_WINDOW":   "120",
		"RATE_LIMIT_REQUESTS_PER_IP": "200",
		"RATE_LIMIT_WINDOW_SECONDS":  "120",
		"CORS_ALLOWED_ORIGINS":       "https://example.com,https://api.example.com",
		"CORS_ALLOWED_METHODS":       "GET,POST,PUT",
		"CORS_ALLOWED_HEADERS":       "Content-Type,Authorization",
		"CORS_EXPOSED_HEADERS":       "X-Total-Count,X-Page-Count",
		"CORS_ALLOW_CREDENTIALS":     "true",
		"CORS_MAX_AGE":               "7200",
	}

	for key, value := range envVars {
		s.Require().NoError(os.Setenv(key, value))
	}

	cfg, err := LoadHttp()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().Equal(EnvProduction, cfg.Environment)
	s.Assert().Equal(logger.LevelError, cfg.Logger.Level)
	s.Assert().Equal(logger.FormatText, cfg.Logger.Format)

	s.Assert().Equal("127.0.0.1", cfg.Server.Host)
	s.Assert().Equal(9090, cfg.Server.Port)
	s.Assert().Equal(60, cfg.Server.ReadTimeout)
	s.Assert().Equal(60, cfg.Server.WriteTimeout)
	s.Assert().Equal(300, cfg.Server.IdleTimeout)

	s.Assert().Equal(2000, cfg.RateLimit.GlobalRequests)
	s.Assert().Equal(120, cfg.RateLimit.GlobalWindow)
	s.Assert().Equal(200, cfg.RateLimit.RequestsPerIP)
	s.Assert().Equal(120, cfg.RateLimit.WindowSeconds)

	s.Assert().Equal([]string{"https://example.com", "https://api.example.com"}, cfg.CORS.AllowedOrigins)
	s.Assert().Equal([]string{"GET", "POST", "PUT"}, cfg.CORS.AllowedMethods)
	s.Assert().Equal([]string{"Content-Type", "Authorization"}, cfg.CORS.AllowedHeaders)
	s.Assert().Equal([]string{"X-Total-Count", "X-Page-Count"}, cfg.CORS.ExposedHeaders)
	s.Assert().True(cfg.CORS.AllowCredentials)
	s.Assert().Equal(7200, cfg.CORS.MaxAge)

	for key := range envVars {
		s.Require().NoError(os.Unsetenv(key))
	}
}

func (s *HttpConfigTestSuite) TestHttpServerConfig_EdgeCases() {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*HttpConfig)
	}{
		{
			name: "zero_port",
			envVars: map[string]string{
				"HTTP_SERVER_PORT": "0",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(0, cfg.Server.Port)
			},
		},
		{
			name: "high_port",
			envVars: map[string]string{
				"HTTP_SERVER_PORT": "65535",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(65535, cfg.Server.Port)
			},
		},
		{
			name: "zero_timeouts",
			envVars: map[string]string{
				"HTTP_SERVER_READ_TIMEOUT":  "0",
				"HTTP_SERVER_WRITE_TIMEOUT": "0",
				"HTTP_SERVER_IDLE_TIMEOUT":  "0",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(0, cfg.Server.ReadTimeout)
				s.Assert().Equal(0, cfg.Server.WriteTimeout)
				s.Assert().Equal(0, cfg.Server.IdleTimeout)
			},
		},
		{
			name: "large_timeouts",
			envVars: map[string]string{
				"HTTP_SERVER_READ_TIMEOUT":  "3600",
				"HTTP_SERVER_WRITE_TIMEOUT": "3600",
				"HTTP_SERVER_IDLE_TIMEOUT":  "7200",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(3600, cfg.Server.ReadTimeout)
				s.Assert().Equal(3600, cfg.Server.WriteTimeout)
				s.Assert().Equal(7200, cfg.Server.IdleTimeout)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for key, value := range tt.envVars {
				s.Require().NoError(os.Setenv(key, value))
			}

			cfg, err := LoadHttp()
			s.Require().NoError(err)
			s.Require().NotNil(cfg)

			tt.check(cfg)

			for key := range tt.envVars {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

func (s *HttpConfigTestSuite) TestRateLimitConfig_EdgeCases() {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*HttpConfig)
	}{
		{
			name: "zero_rate_limits",
			envVars: map[string]string{
				"RATE_LIMIT_GLOBAL_REQUESTS": "0",
				"RATE_LIMIT_GLOBAL_WINDOW":   "0",
				"RATE_LIMIT_REQUESTS_PER_IP": "0",
				"RATE_LIMIT_WINDOW_SECONDS":  "0",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(0, cfg.RateLimit.GlobalRequests)
				s.Assert().Equal(0, cfg.RateLimit.GlobalWindow)
				s.Assert().Equal(0, cfg.RateLimit.RequestsPerIP)
				s.Assert().Equal(0, cfg.RateLimit.WindowSeconds)
			},
		},
		{
			name: "high_rate_limits",
			envVars: map[string]string{
				"RATE_LIMIT_GLOBAL_REQUESTS": "1000000",
				"RATE_LIMIT_GLOBAL_WINDOW":   "3600",
				"RATE_LIMIT_REQUESTS_PER_IP": "10000",
				"RATE_LIMIT_WINDOW_SECONDS":  "3600",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(1000000, cfg.RateLimit.GlobalRequests)
				s.Assert().Equal(3600, cfg.RateLimit.GlobalWindow)
				s.Assert().Equal(10000, cfg.RateLimit.RequestsPerIP)
				s.Assert().Equal(3600, cfg.RateLimit.WindowSeconds)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for key, value := range tt.envVars {
				s.Require().NoError(os.Setenv(key, value))
			}

			cfg, err := LoadHttp()
			s.Require().NoError(err)
			s.Require().NotNil(cfg)

			tt.check(cfg)

			for key := range tt.envVars {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

func (s *HttpConfigTestSuite) TestCORSConfig_EdgeCases() {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*HttpConfig)
	}{
		{
			name: "empty_cors_arrays",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "",
				"CORS_ALLOWED_METHODS": "",
				"CORS_ALLOWED_HEADERS": "",
				"CORS_EXPOSED_HEADERS": "",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Empty(cfg.CORS.AllowedOrigins)
				s.Assert().Empty(cfg.CORS.AllowedMethods)
				s.Assert().Empty(cfg.CORS.AllowedHeaders)
				s.Assert().Empty(cfg.CORS.ExposedHeaders)
			},
		},
		{
			name: "single_values_in_arrays",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "https://single-origin.com",
				"CORS_ALLOWED_METHODS": "GET",
				"CORS_ALLOWED_HEADERS": "Content-Type",
				"CORS_EXPOSED_HEADERS": "X-Total-Count",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal([]string{"https://single-origin.com"}, cfg.CORS.AllowedOrigins)
				s.Assert().Equal([]string{"GET"}, cfg.CORS.AllowedMethods)
				s.Assert().Equal([]string{"Content-Type"}, cfg.CORS.AllowedHeaders)
				s.Assert().Equal([]string{"X-Total-Count"}, cfg.CORS.ExposedHeaders)
			},
		},
		{
			name: "multiple_values_with_spaces",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "https://example.com, https://api.example.com , https://admin.example.com",
				"CORS_ALLOWED_METHODS": "GET, POST , PUT,DELETE",
				"CORS_ALLOWED_HEADERS": "Content-Type , Authorization,X-API-Key",
			},
			check: func(cfg *HttpConfig) {
				expectedOrigins := []string{"https://example.com", " https://api.example.com ", " https://admin.example.com"}
				expectedMethods := []string{"GET", " POST ", " PUT", "DELETE"}
				expectedHeaders := []string{"Content-Type ", " Authorization", "X-API-Key"}

				s.Assert().Equal(expectedOrigins, cfg.CORS.AllowedOrigins)
				s.Assert().Equal(expectedMethods, cfg.CORS.AllowedMethods)
				s.Assert().Equal(expectedHeaders, cfg.CORS.AllowedHeaders)
			},
		},
		{
			name: "boolean_string_variations",
			envVars: map[string]string{
				"CORS_ALLOW_CREDENTIALS": "1",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().True(cfg.CORS.AllowCredentials)
			},
		},
		{
			name: "negative_max_age",
			envVars: map[string]string{
				"CORS_MAX_AGE": "-1",
			},
			check: func(cfg *HttpConfig) {
				s.Assert().Equal(-1, cfg.CORS.MaxAge)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for key, value := range tt.envVars {
				s.Require().NoError(os.Setenv(key, value))
			}

			cfg, err := LoadHttp()
			s.Require().NoError(err)
			s.Require().NotNil(cfg)

			tt.check(cfg)

			for key := range tt.envVars {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

func (s *HttpConfigTestSuite) TestHttpConfig_InheritsBaseConfig() {
	s.Require().NoError(os.Setenv("ENV", EnvStaging))
	defer func() { s.Require().NoError(os.Unsetenv("ENV")) }()

	cfg, err := LoadHttp()

	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Assert().True(cfg.IsStaging())
	s.Assert().False(cfg.IsProduction())
	s.Assert().False(cfg.IsDevelopment())
	s.Assert().False(cfg.IsTest())
}

func (s *HttpConfigTestSuite) TestZeroValues() {
	cfg := HttpConfig{}

	s.Assert().Equal("", cfg.Environment)
	s.Assert().Equal("", cfg.Server.Host)
	s.Assert().Equal(0, cfg.Server.Port)
	s.Assert().Equal(0, cfg.Server.ReadTimeout)
	s.Assert().Equal(0, cfg.Server.WriteTimeout)
	s.Assert().Equal(0, cfg.Server.IdleTimeout)

	s.Assert().Equal(0, cfg.RateLimit.GlobalRequests)
	s.Assert().Equal(0, cfg.RateLimit.GlobalWindow)
	s.Assert().Equal(0, cfg.RateLimit.RequestsPerIP)
	s.Assert().Equal(0, cfg.RateLimit.WindowSeconds)

	s.Assert().Nil(cfg.CORS.AllowedOrigins)
	s.Assert().Nil(cfg.CORS.AllowedMethods)
	s.Assert().Nil(cfg.CORS.AllowedHeaders)
	s.Assert().Nil(cfg.CORS.ExposedHeaders)
	s.Assert().False(cfg.CORS.AllowCredentials)
	s.Assert().Equal(0, cfg.CORS.MaxAge)
}

func (s *HttpConfigTestSuite) TestLoadHttp_Performance() {
	for i := 0; i < 100; i++ {
		cfg, err := LoadHttp()
		s.Require().NoError(err)
		s.Require().NotNil(cfg)
	}
}

func (s *HttpConfigTestSuite) TestCORSArrayParsing() {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"single", []string{"single"}},
		{"one,two,three", []string{"one", "two", "three"}},
		{"*", []string{"*"}},
		{"value1,value2,value3,value4", []string{"value1", "value2", "value3", "value4"}},
	}

	for _, tc := range testCases {
		s.Run("cors_parsing_"+strings.ReplaceAll(tc.input, ",", "_"), func() {
			s.Require().NoError(os.Setenv("CORS_ALLOWED_ORIGINS", tc.input))
			defer func() {
				s.Require().NoError(os.Unsetenv("CORS_ALLOWED_ORIGINS"))
			}()

			cfg, err := LoadHttp()
			s.Require().NoError(err)

			if tc.input == "" {
				s.Assert().Empty(cfg.CORS.AllowedOrigins)
			} else {
				s.Assert().Equal(tc.expected, cfg.CORS.AllowedOrigins)
			}
		})
	}
}

func BenchmarkLoadHttp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := LoadHttp()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadHttp_WithEnvVars(b *testing.B) {
	_ = os.Setenv("HTTP_SERVER_PORT", "9090")
	_ = os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com,https://api.example.com")
	defer func() {
		_ = os.Unsetenv("HTTP_SERVER_PORT")
		_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadHttp()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestHttpConfigTestSuite(t *testing.T) {
	suite.Run(t, new(HttpConfigTestSuite))
}
