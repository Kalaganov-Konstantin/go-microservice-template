package middleware

import (
	"microservice/internal/platform/logger"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(baseLogger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			reqID := middleware.GetReqID(r.Context())
			contextLogger := baseLogger.With(logger.String("request_id", reqID))
			ctx := logger.WithLogger(r.Context(), contextLogger)

			next.ServeHTTP(ww, r.WithContext(ctx))

			contextLogger.Info("HTTP Request",
				logger.String("method", r.Method),
				logger.String("path", r.URL.Path),
				logger.String("remote_addr", r.RemoteAddr),
				logger.Int("status", ww.Status()),
				logger.String("duration", time.Since(start).String()),
			)
		})
	}
}
