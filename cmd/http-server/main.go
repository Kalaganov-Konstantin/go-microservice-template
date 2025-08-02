package main

import (
	"microservice/internal/adapters/database"
	"microservice/internal/adapters/health"
	httpAdapter "microservice/internal/adapters/http"
	exampleHandler "microservice/internal/adapters/http/example"
	healthHttp "microservice/internal/adapters/http/health"
	exampleRepo "microservice/internal/adapters/repository/postgres"
	"microservice/internal/adapters/validator"
	"microservice/internal/config"
	exampleDomain "microservice/internal/core/domain/example"
	"microservice/internal/core/ports"
	exampleUseCase "microservice/internal/core/usecase/example"
	"microservice/internal/platform/database/postgres"
	platformHealth "microservice/internal/platform/health"
	"microservice/internal/platform/logger"
	"microservice/internal/platform/metrics"
	"microservice/internal/version"

	"go.uber.org/fx"
)

func main() {
	fx.New(appModule).Run()
}

var appModule = fx.Options(
	// Platform
	fx.Provide(config.LoadBase),
	fx.Provide(config.LoadHttp),
	fx.Provide(config.LoadDatabase),
	fx.Provide(func(cfg *config.BaseConfig) logger.Config {
		return logger.Config{
			Environment: cfg.Environment,
			Level:       cfg.Logger.Level,
			Format:      cfg.Logger.Format,
		}
	}),
	fx.Provide(logger.NewZapLogger),
	fx.Provide(validator.NewPlaygroundAdapter),
	fx.Provide(postgres.New),
	fx.Provide(database.NewDatabaseLifecycle),

	// Health Checks
	fx.Provide(fx.Annotate(health.NewMemoryChecker, fx.As(new(platformHealth.Checker)), fx.ResultTags(`group:"health_checkers"`))),
	fx.Provide(fx.Annotate(
		func(db *database.Lifecycle) *health.DatabaseChecker {
			return health.NewDatabaseChecker(db, "postgres")
		},
		fx.As(new(platformHealth.Checker)),
		fx.ResultTags(`group:"health_checkers"`),
	)),
	fx.Provide(fx.Annotate(
		func(checkers []platformHealth.Checker) *platformHealth.Manager {
			m := platformHealth.NewManager()
			for _, checker := range checkers {
				m.Register(checker)
			}
			return m
		},
		fx.ParamTags(`group:"health_checkers"`),
		fx.As(new(platformHealth.ManagerInterface)),
	)),

	// HTTP Server
	fx.Provide(metrics.NewProvider),
	fx.Provide(httpAdapter.NewServer),
	fx.Provide(httpAdapter.NewRouter),
	fx.Provide(exampleHandler.NewHandler),
	fx.Provide(func() *healthHttp.LivenessHandler {
		return healthHttp.NewLivenessHandler(version.Get())
	}),
	fx.Provide(func(hm platformHealth.ManagerInterface) *healthHttp.ReadinessHandler {
		return healthHttp.NewReadinessHandler(version.Get(), hm)
	}),
	fx.Provide(func(cfg *config.HttpConfig, log logger.Logger, example *exampleHandler.Handler, liveness *healthHttp.LivenessHandler, readiness *healthHttp.ReadinessHandler, metrics *metrics.Provider) httpAdapter.RouterDependencies {
		return httpAdapter.RouterDependencies{
			Config:           cfg,
			Logger:           log,
			ExampleHandler:   example,
			LivenessHandler:  liveness,
			ReadinessHandler: readiness,
			MetricsProvider:  metrics,
		}
	}),

	// Domain
	fx.Provide(fx.Annotate(exampleRepo.NewRepository, fx.As(new(ports.ExampleRepository)))),
	fx.Provide(fx.Annotate(exampleDomain.NewService, fx.As(new(exampleUseCase.EntityChecker)))),
	fx.Provide(fx.Annotate(exampleUseCase.NewUsecase, fx.As(new(exampleHandler.Manager)))),

	// Lifecycle Hooks
	fx.Invoke(func(lc fx.Lifecycle, db *database.Lifecycle, srv *httpAdapter.Server) {
		lc.Append(fx.Hook{
			OnStart: db.Start,
			OnStop:  db.Stop,
		})
		lc.Append(fx.Hook{
			OnStart: srv.Start,
			OnStop:  srv.Stop,
		})
	}),

	//fx.NopLogger,
)
