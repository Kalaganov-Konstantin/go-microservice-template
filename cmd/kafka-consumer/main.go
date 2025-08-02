package main

import (
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// TODO:
		// ConfigModule,
		// ConsumerModule,
		// MetricsModule,

		fx.NopLogger,
	).Run()
}
