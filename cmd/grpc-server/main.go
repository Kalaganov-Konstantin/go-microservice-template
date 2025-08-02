package main

import (
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// TODO:
		// ConfigModule,
		// ServerModule,
		// HealthModule,

		fx.NopLogger,
	).Run()
}
