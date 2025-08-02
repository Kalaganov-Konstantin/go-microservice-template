package http

import (
	"errors"
	httpErrors "microservice/internal/platform/http"
	"microservice/internal/platform/logger"
	"net/http"

	"microservice/internal/adapters/http/response"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func ErrorHandler(next HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := next(w, r)
		if err == nil {
			return
		}

		contextLogger := logger.FromContext(r.Context())

		var httpErr *httpErrors.Error
		if errors.As(err, &httpErr) {
			response.RespondError(w, httpErr.StatusCode, httpErr)
			return
		}

		contextLogger.Error("Unexpected server error",
			logger.String("method", r.Method),
			logger.String("path", r.URL.Path),
			logger.String("remote_addr", r.RemoteAddr),
			logger.Error(err))
		response.RespondError(w, http.StatusInternalServerError, errors.New("internal server error"))
	}
}
