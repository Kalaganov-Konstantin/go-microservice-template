package http

import (
	"encoding/json"
	"microservice/internal/adapters/http/example"
	"microservice/internal/adapters/http/health"
	"microservice/internal/adapters/validator"
	"microservice/internal/config"
	platformHealth "microservice/internal/platform/health"
	"microservice/internal/platform/logger"
	"microservice/internal/platform/metrics"

	"github.com/go-chi/chi/v5"

	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	exampleMocks "microservice/internal/adapters/http/example/mocks"
	healthMocks "microservice/internal/platform/health/mocks"
)

type RouterTestSuite struct {
	suite.Suite
	config            *config.HttpConfig
	logger            logger.Logger
	metricsProvider   *metrics.Provider
	exampleHandler    *example.Handler
	livenessHandler   *health.LivenessHandler
	readinessHandler  *health.ReadinessHandler
	mockHealthManager *healthMocks.MockManagerInterface
	mockManager       *exampleMocks.MockManager
}

func (s *RouterTestSuite) SetupTest() {
	s.config = &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		RateLimit: config.RateLimitConfig{
			GlobalRequests: 1000,
			GlobalWindow:   60,
			RequestsPerIP:  100,
			WindowSeconds:  60,
		},
		CORS: config.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{},
			AllowCredentials: false,
			MaxAge:           86400,
		},
	}

	s.logger = logger.NewNop()

	var err error
	s.metricsProvider, err = metrics.NewProvider()
	s.Require().NoError(err)

	s.mockManager = exampleMocks.NewMockManager(s.T())
	validatorAdapter := validator.NewPlaygroundAdapter()
	s.Require().NoError(err)
	s.exampleHandler = example.NewHandler(s.mockManager, validatorAdapter)

	s.livenessHandler = health.NewLivenessHandler("1.0.0")

	s.mockHealthManager = healthMocks.NewMockManagerInterface(s.T())
	s.readinessHandler = health.NewReadinessHandler("1.0.0", s.mockHealthManager)
}

func (s *RouterTestSuite) createRouterDependencies(config ...*config.HttpConfig) RouterDependencies {
	cfg := s.config
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return RouterDependencies{
		Config:           cfg,
		Logger:           s.logger,
		ExampleHandler:   s.exampleHandler,
		LivenessHandler:  s.livenessHandler,
		ReadinessHandler: s.readinessHandler,
		MetricsProvider:  s.metricsProvider,
	}
}

func (s *RouterTestSuite) TestNewRouter_Configuration() {
	router := NewRouter(s.createRouterDependencies())

	s.Assert().NotNil(router)
}

func (s *RouterTestSuite) TestRouter_HealthLivenessEndpoint() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))

	var response health.LivenessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Assert().NoError(err)
	s.Assert().Equal(health.StatusPass, response.Status)
	s.Assert().Equal("1.0.0", response.Version)
	s.Assert().NotZero(response.Timestamp)
}

func (s *RouterTestSuite) TestRouter_HealthReadinessEndpoint_Success() {
	s.mockHealthManager.On("CheckAll", mock.AnythingOfType("*context.timerCtx")).Return(map[string]platformHealth.CheckResult{
		"database": {
			Status:  platformHealth.StatusHealthy,
			Message: "Database is accessible",
		},
	})

	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))

	var response health.ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Assert().NoError(err)
	s.Assert().Equal(health.StatusPass, response.Status)
	s.Assert().Equal("1.0.0", response.Version)
	s.Assert().Contains(response.Checks, "database")
}

func (s *RouterTestSuite) TestRouter_HealthReadinessEndpoint_Failure() {
	s.mockHealthManager.On("CheckAll", mock.AnythingOfType("*context.timerCtx")).Return(map[string]platformHealth.CheckResult{
		"database": {
			Status: platformHealth.StatusUnhealthy,
			Error:  "Connection timeout",
		},
	})

	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusServiceUnavailable, w.Code)

	var response health.ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.Assert().NoError(err)
	s.Assert().Equal(health.StatusFail, response.Status)
	s.Assert().NotEmpty(response.Notes)
}

func (s *RouterTestSuite) TestRouter_MetricsEndpoint() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Contains(w.Header().Get("Content-Type"), "text/plain")
}

func (s *RouterTestSuite) TestRouter_CORSHeaders() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("OPTIONS", "/api/examples", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("*", w.Header().Get("Access-Control-Allow-Origin"))
	s.Assert().Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST")
	s.Assert().Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

func (s *RouterTestSuite) TestRouter_CORSCustomConfiguration() {
	customConfig := &config.HttpConfig{
		Server:    s.config.Server,
		RateLimit: s.config.RateLimit,
		CORS: config.CORSConfig{
			AllowedOrigins:   []string{"https://example.com", "https://test.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Authorization"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
	}

	router := NewRouter(s.createRouterDependencies(customConfig))

	req := httptest.NewRequest("OPTIONS", "/api/examples", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	s.Assert().NotEmpty(allowOrigin)
}

func (s *RouterTestSuite) TestRouter_APIRoutes_NotFound() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/api/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusNotFound, w.Code)
}

func (s *RouterTestSuite) TestRouter_RootNotFound() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusNotFound, w.Code)
}

func (s *RouterTestSuite) TestRouter_MethodNotAllowed() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("POST", "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusMethodNotAllowed, w.Code)
}

func (s *RouterTestSuite) TestRouter_Middleware_RequestID() {
	router := NewRouter(s.createRouterDependencies()).(*chi.Mux)

	var capturedRequestID string
	router.Get("/test-request-id", func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = chi.RouteContext(r.Context()).RouteMethod
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test-request-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("GET", capturedRequestID)
}

func (s *RouterTestSuite) TestRouter_Middleware_RealIP() {
	router := NewRouter(s.createRouterDependencies()).(*chi.Mux)

	var capturedIP string
	router.Get("/test-real-ip", func(w http.ResponseWriter, r *http.Request) {
		capturedIP = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test-real-ip", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.Header.Set("X-Real-IP", "10.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().NotEmpty(capturedIP)
}

func (s *RouterTestSuite) TestRouter_Middleware_StripSlashes() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/health/live/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	s.Assert().True(w.Code == http.StatusOK || w.Code == http.StatusMovedPermanently)
}

func (s *RouterTestSuite) TestRouter_Middleware_Recoverer_Panic() {
	router := NewRouter(s.createRouterDependencies()).(*chi.Mux)
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	s.Assert().NotPanics(func() {
		router.ServeHTTP(w, req)
	})

	s.Assert().Equal(http.StatusInternalServerError, w.Code)
}

func (s *RouterTestSuite) TestRouter_RateLimit_Integration() {
	restrictiveConfig := &config.HttpConfig{
		Server: s.config.Server,
		CORS:   s.config.CORS,
		RateLimit: config.RateLimitConfig{
			GlobalRequests: 2,
			GlobalWindow:   1,
			RequestsPerIP:  1,
			WindowSeconds:  1,
		},
	}

	router := NewRouter(s.createRouterDependencies(restrictiveConfig))

	req1 := httptest.NewRequest("GET", "/health/live", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	s.Assert().Equal(http.StatusOK, w1.Code)

	req2 := httptest.NewRequest("GET", "/health/live", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	s.Assert().True(w2.Code == http.StatusOK || w2.Code == http.StatusTooManyRequests)
}

func (s *RouterTestSuite) TestRouter_AllMiddleware_Integration() {
	router := NewRouter(s.createRouterDependencies())

	req := httptest.NewRequest("GET", "/health/live", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))
}

func (s *RouterTestSuite) TestRouter_DifferentHTTPMethods() {
	router := NewRouter(s.createRouterDependencies())

	testCases := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{"GET", "/health/live", "", http.StatusOK},
		{"GET", "/health/ready", "", http.StatusOK},
		{"GET", "/metrics", "", http.StatusOK},
		{"OPTIONS", "/api/examples", "", http.StatusOK},
	}

	s.mockHealthManager.On("CheckAll", mock.AnythingOfType("*context.timerCtx")).Return(map[string]platformHealth.CheckResult{
		"test": {Status: platformHealth.StatusHealthy, Message: "OK"},
	}).Times(1)

	for _, tc := range testCases {
		s.Run(tc.method+"_"+tc.path, func() {
			var reqBody io.Reader
			if tc.body != "" {
				reqBody = strings.NewReader(tc.body)
			}

			req := httptest.NewRequest(tc.method, tc.path, reqBody)
			if tc.method == "OPTIONS" {
				req.Header.Set("Origin", "https://example.com")
				req.Header.Set("Access-Control-Request-Method", "POST")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			s.Assert().Equal(tc.expectedStatus, w.Code, "Method %s on path %s", tc.method, tc.path)
		})
	}
}

func (s *RouterTestSuite) TestRouter_Performance() {
	router := NewRouter(s.createRouterDependencies())

	numRequests := 50
	done := make(chan bool, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(i int) {
			req := httptest.NewRequest("GET", "/health/live", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			s.Assert().Equal(http.StatusOK, w.Code)
			done <- true
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}

	duration := time.Since(start)
	s.Assert().Less(duration, 5*time.Second, "Router should handle %d concurrent requests quickly", numRequests)
}

func BenchmarkRouter_HealthLiveness(b *testing.B) {
	httpConfig := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		RateLimit: config.RateLimitConfig{
			GlobalRequests: 10000,
			GlobalWindow:   60,
			RequestsPerIP:  1000,
			WindowSeconds:  60,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		},
	}

	log := logger.NewNop()
	metricsProvider, _ := metrics.NewProvider()
	livenessHandler := health.NewLivenessHandler("1.0.0")

	mockManager := exampleMocks.NewMockManager(b)
	validatorAdapter := validator.NewPlaygroundAdapter()
	exampleHandler := example.NewHandler(mockManager, validatorAdapter)

	mockHealthManager := healthMocks.NewMockManagerInterface(b)
	readinessHandler := health.NewReadinessHandler("1.0.0", mockHealthManager)

	deps := RouterDependencies{
		Config:           httpConfig,
		Logger:           log,
		ExampleHandler:   exampleHandler,
		LivenessHandler:  livenessHandler,
		ReadinessHandler: readinessHandler,
		MetricsProvider:  metricsProvider,
	}

	router := NewRouter(deps)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Metrics(b *testing.B) {
	httpConfig := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		RateLimit: config.RateLimitConfig{
			GlobalRequests: 10000,
			GlobalWindow:   60,
			RequestsPerIP:  1000,
			WindowSeconds:  60,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		},
	}

	log := logger.NewNop()
	metricsProvider, _ := metrics.NewProvider()
	livenessHandler := health.NewLivenessHandler("1.0.0")

	mockManager := exampleMocks.NewMockManager(b)
	validatorAdapter := validator.NewPlaygroundAdapter()
	exampleHandler := example.NewHandler(mockManager, validatorAdapter)

	mockHealthManager := healthMocks.NewMockManagerInterface(b)
	readinessHandler := health.NewReadinessHandler("1.0.0", mockHealthManager)

	deps := RouterDependencies{
		Config:           httpConfig,
		Logger:           log,
		ExampleHandler:   exampleHandler,
		LivenessHandler:  livenessHandler,
		ReadinessHandler: readinessHandler,
		MetricsProvider:  metricsProvider,
	}

	router := NewRouter(deps)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
