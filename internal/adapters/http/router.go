package http

import (
	"microservice/internal/platform/logger"
	"microservice/internal/platform/metrics"
	platformMiddleware "microservice/internal/platform/middleware"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"microservice/internal/adapters/http/example"
	"microservice/internal/adapters/http/health"
	"microservice/internal/config"
)

type RouterDependencies struct {
	Config           *config.HttpConfig
	Logger           logger.Logger
	ExampleHandler   *example.Handler
	LivenessHandler  *health.LivenessHandler
	ReadinessHandler *health.ReadinessHandler
	MetricsProvider  *metrics.Provider
}

func NewRouter(deps RouterDependencies) http.Handler {
	cfg := deps.Config
	log := deps.Logger
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(platformMiddleware.RequestLogger(log))
	r.Use(platformMiddleware.MetricsMiddleware(deps.MetricsProvider))
	r.Use(platformMiddleware.Recovery(log))
	r.Use(middleware.StripSlashes)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	r.Use(httprate.LimitAll(
		cfg.RateLimit.GlobalRequests,
		time.Duration(cfg.RateLimit.GlobalWindow)*time.Second,
	))
	r.Use(httprate.LimitByIP(
		cfg.RateLimit.RequestsPerIP,
		time.Duration(cfg.RateLimit.WindowSeconds)*time.Second,
	))

	r.Get("/health/live", deps.LivenessHandler.Check)
	r.Get("/health/ready", deps.ReadinessHandler.Check)

	r.Handle("/metrics", deps.MetricsProvider.Handler())

	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Route("/examples", func(exampleRouter chi.Router) {
			exampleRouter.Post("/", ErrorHandler(deps.ExampleHandler.CreateEntity))
			exampleRouter.Get("/{id}", ErrorHandler(deps.ExampleHandler.GetEntity))
		})
	})

	return r
}
