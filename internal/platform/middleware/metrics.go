package middleware

import (
	"microservice/internal/platform/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func MetricsMiddleware(metricsProvider *metrics.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			start := time.Now()

			metricsProvider.RequestsInFlight.Add(ctx, 1)
			defer metricsProvider.RequestsInFlight.Add(ctx, -1)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(ww.Status())
			method := r.Method
			path := r.URL.Path

			metricsProvider.RequestsTotal.Add(ctx, 1,
				metric.WithAttributes(
					attribute.String("method", method),
					attribute.String("path", path),
					attribute.String("status", status),
				),
			)

			metricsProvider.RequestDuration.Record(ctx, duration,
				metric.WithAttributes(
					attribute.String("method", method),
					attribute.String("path", path),
					attribute.String("status", status),
				),
			)
		})
	}
}
