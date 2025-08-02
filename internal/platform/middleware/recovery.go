package middleware

import (
	"fmt"
	"microservice/internal/platform/logger"
	"net/http"
	"runtime/debug"
)

func Recovery(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					contextLogger := logger.FromContext(r.Context())
					if contextLogger == nil {
						contextLogger = log
					}

					contextLogger.Error("Panic recovered",
						logger.String("method", r.Method),
						logger.String("url", r.URL.Path),
						logger.String("remote_addr", r.RemoteAddr),
						logger.String("user_agent", r.UserAgent()),
						logger.String("panic", fmt.Sprintf("%v", err)),
						logger.String("stack", string(debug.Stack())),
					)

					w.Header().Set("Connection", "close")

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
