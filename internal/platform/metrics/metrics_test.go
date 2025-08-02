package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MetricsTestSuite struct {
	suite.Suite
	provider *Provider
}

func (s *MetricsTestSuite) SetupTest() {
	var err error
	s.provider, err = NewProvider()
	s.Require().NoError(err)
	s.Require().NotNil(s.provider)
}

func (s *MetricsTestSuite) TestNewProvider_Success() {
	provider, err := NewProvider()

	s.Assert().NoError(err)
	s.Assert().NotNil(provider)
	s.Assert().NotNil(provider.RequestsTotal)
	s.Assert().NotNil(provider.RequestDuration)
	s.Assert().NotNil(provider.RequestsInFlight)
	s.Assert().NotNil(provider.registry)
}

func (s *MetricsTestSuite) TestNewProvider_MultipleProviders() {
	provider1, err1 := NewProvider()
	s.Assert().NoError(err1)
	s.Assert().NotNil(provider1)

	provider2, err2 := NewProvider()
	s.Assert().NoError(err2)
	s.Assert().NotNil(provider2)

	s.Assert().NotEqual(provider1, provider2)
	s.Assert().NotEqual(provider1.registry, provider2.registry)
}

func (s *MetricsTestSuite) TestProvider_Handler() {
	handler := s.provider.Handler()

	s.Assert().NotNil(handler)

	ctx := context.Background()
	s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "GET")))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Contains(w.Header().Get("Content-Type"), "text/plain")

	body := w.Body.String()
	s.Assert().NotEmpty(body)

	s.Assert().True(strings.Contains(body, "# HELP") || strings.Contains(body, "# TYPE"))
}

func (s *MetricsTestSuite) TestProvider_RequestsTotal_Counter() {
	ctx := context.Background()

	s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "GET"), attribute.String("status", "200")))
	s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "POST"), attribute.String("status", "201")))
	s.provider.RequestsTotal.Add(ctx, 5, metric.WithAttributes(attribute.String("method", "GET"), attribute.String("status", "404")))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_total")
}

func (s *MetricsTestSuite) TestProvider_RequestDuration_Histogram() {
	ctx := context.Background()

	s.provider.RequestDuration.Record(ctx, 0.1, metric.WithAttributes(attribute.String("method", "GET"), attribute.String("status", "200")))
	s.provider.RequestDuration.Record(ctx, 0.5, metric.WithAttributes(attribute.String("method", "POST"), attribute.String("status", "201")))
	s.provider.RequestDuration.Record(ctx, 1.2, metric.WithAttributes(attribute.String("method", "GET"), attribute.String("status", "500")))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	s.Assert().Contains(body, "http_request_duration_seconds")
}

func (s *MetricsTestSuite) TestProvider_RequestsInFlight_UpDownCounter() {
	ctx := context.Background()

	s.provider.RequestsInFlight.Add(ctx, 5)
	s.provider.RequestsInFlight.Add(ctx, -2)
	s.provider.RequestsInFlight.Add(ctx, 3)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_in_flight")
}

func (s *MetricsTestSuite) TestProvider_MultipleMetricsWithAttributes() {
	ctx := context.Background()

	testCases := []struct {
		method   string
		status   string
		duration float64
		count    int64
	}{
		{"GET", "200", 0.1, 1},
		{"GET", "404", 0.05, 1},
		{"POST", "201", 0.3, 1},
		{"POST", "400", 0.2, 1},
		{"PUT", "200", 0.4, 1},
		{"DELETE", "204", 0.1, 1},
	}

	for _, tc := range testCases {
		attrs := attribute.NewSet(
			attribute.String("method", tc.method),
			attribute.String("status", tc.status),
		)

		s.provider.RequestsTotal.Add(ctx, tc.count, metric.WithAttributeSet(attrs))
		s.provider.RequestDuration.Record(ctx, tc.duration, metric.WithAttributeSet(attrs))
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_total")
	s.Assert().Contains(body, "http_request_duration_seconds")

	s.Assert().Contains(body, "GET")
	s.Assert().Contains(body, "POST")
	s.Assert().Contains(body, "PUT")
	s.Assert().Contains(body, "DELETE")
}

func (s *MetricsTestSuite) TestProvider_ConcurrentMetrics() {
	ctx := context.Background()
	numGoroutines := 10
	requestsPerGoroutine := 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()

			for j := 0; j < requestsPerGoroutine; j++ {
				attrs := attribute.NewSet(
					attribute.String("method", "GET"),
					attribute.String("status", "200"),
					attribute.Int("routine", routineID),
				)

				s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
				s.provider.RequestDuration.Record(ctx, 0.1, metric.WithAttributeSet(attrs))
				s.provider.RequestsInFlight.Add(ctx, 1)
				s.provider.RequestsInFlight.Add(ctx, -1)
			}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)
	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_total")
}

func (s *MetricsTestSuite) TestProvider_HandlerWithDifferentMethods() {

	ctx := context.Background()
	s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "GET")))

	handler := s.provider.Handler()

	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		s.Assert().Equal(http.StatusOK, w.Code, "Method %s should work", method)

		if method != "HEAD" {
			body := w.Body.String()
			s.Assert().NotEmpty(body, "Method %s should return metrics", method)
		}
	}
}

func (s *MetricsTestSuite) TestProvider_MetricsFormat() {
	ctx := context.Background()

	s.provider.RequestsTotal.Add(ctx, 10,
		metric.WithAttributes(
			attribute.String("method", "GET"),
			attribute.String("status", "200"),
			attribute.String("path", "/api/test")))

	s.provider.RequestDuration.Record(ctx, 0.25,
		metric.WithAttributes(
			attribute.String("method", "GET"),
			attribute.String("status", "200")))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	s.Assert().Contains(body, "http_requests_total")
	s.Assert().Contains(body, "http_request_duration_seconds")

	lines := strings.Split(body, "\n")
	hasHelpLine := false
	hasTypeLine := false

	for _, line := range lines {
		if strings.Contains(line, "# HELP") {
			hasHelpLine = true
		}
		if strings.Contains(line, "# TYPE") {
			hasTypeLine = true
		}
	}

	s.Assert().True(hasHelpLine || hasTypeLine, "Metrics should have HELP or TYPE annotations")
}

func (s *MetricsTestSuite) TestProvider_LargeNumberOfMetrics() {
	ctx := context.Background()

	numMetrics := 1000

	start := time.Now()

	for i := 0; i < numMetrics; i++ {
		attrs := attribute.NewSet(
			attribute.String("method", "GET"),
			attribute.String("status", "200"),
			attribute.Int("request_id", i),
		)

		s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
		s.provider.RequestDuration.Record(ctx, 0.1, metric.WithAttributeSet(attrs))
	}

	recordingDuration := time.Since(start)

	s.Assert().Less(recordingDuration, 1*time.Second, "Recording %d metrics should be fast", numMetrics)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exportStart := time.Now()
	s.provider.Handler().ServeHTTP(w, req)
	exportDuration := time.Since(exportStart)

	s.Assert().Equal(http.StatusOK, w.Code)
	s.Assert().Less(exportDuration, 5*time.Second, "Exporting metrics should be reasonably fast")
}

func (s *MetricsTestSuite) TestProvider_MetricsWithSpecialCharacters() {
	ctx := context.Background()

	testAttrs := []attribute.Set{
		attribute.NewSet(
			attribute.String("method", "GET"),
			attribute.String("path", "/api/user/123"),
			attribute.String("status", "200"),
		),
		attribute.NewSet(
			attribute.String("method", "POST"),
			attribute.String("path", "/api/user"),
			attribute.String("status", "201"),
		),
		attribute.NewSet(
			attribute.String("method", "GET"),
			attribute.String("path", "/health/ready"),
			attribute.String("status", "503"),
		),
	}

	for _, attrs := range testAttrs {
		s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
		s.provider.RequestDuration.Record(ctx, 0.1, metric.WithAttributeSet(attrs))
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	s.provider.Handler().ServeHTTP(w, req)

	s.Assert().Equal(http.StatusOK, w.Code)

	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_total")
	s.Assert().Contains(body, "http_request_duration_seconds")
}

func (s *MetricsTestSuite) TestProvider_ResetBetweenTests() {

	ctx := context.Background()
	s.provider.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("test", "isolation")))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	s.provider.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	s.Assert().Contains(body, "http_requests_total")

	s.Assert().NotNil(s.provider)
}

func (s *MetricsTestSuite) TestProvider_Performance() {
	ctx := context.Background()
	numOperations := 10000

	start := time.Now()

	for i := 0; i < numOperations; i++ {
		s.provider.RequestsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", "GET"),
				attribute.String("status", "200")))

		s.provider.RequestDuration.Record(ctx, 0.1,
			metric.WithAttributes(
				attribute.String("method", "GET"),
				attribute.String("status", "200")))

		s.provider.RequestsInFlight.Add(ctx, 1)
		s.provider.RequestsInFlight.Add(ctx, -1)
	}

	duration := time.Since(start)

	s.Assert().Less(duration, 1*time.Second, "Should record %d operations quickly", numOperations)

	opsPerSecond := float64(numOperations*3) / duration.Seconds()
	s.Assert().Greater(opsPerSecond, 10000.0, "Should achieve reasonable ops/second")
}

func BenchmarkProvider_RequestsTotal(b *testing.B) {
	provider, err := NewProvider()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.RequestsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("method", "GET"),
				attribute.String("status", "200")))
	}
}

func BenchmarkProvider_RequestDuration(b *testing.B) {
	provider, err := NewProvider()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.RequestDuration.Record(ctx, 0.1,
			metric.WithAttributes(
				attribute.String("method", "GET"),
				attribute.String("status", "200")))
	}
}

func BenchmarkProvider_RequestsInFlight(b *testing.B) {
	provider, err := NewProvider()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.RequestsInFlight.Add(ctx, 1)
		provider.RequestsInFlight.Add(ctx, -1)
	}
}

func BenchmarkProvider_Handler(b *testing.B) {
	provider, err := NewProvider()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	provider.RequestsTotal.Add(ctx, 100, metric.WithAttributes(attribute.String("method", "GET")))
	provider.RequestDuration.Record(ctx, 0.1, metric.WithAttributes(attribute.String("method", "GET")))

	handler := provider.Handler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkProvider_ConcurrentMetrics(b *testing.B) {
	provider, err := NewProvider()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			provider.RequestsTotal.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("method", "GET"),
					attribute.String("status", "200")))
			provider.RequestDuration.Record(ctx, 0.1,
				metric.WithAttributes(
					attribute.String("method", "GET"),
					attribute.String("status", "200")))
			provider.RequestsInFlight.Add(ctx, 1)
			provider.RequestsInFlight.Add(ctx, -1)
		}
	})
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}
