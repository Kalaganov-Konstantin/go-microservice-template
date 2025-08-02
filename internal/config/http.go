package config

import (
	"github.com/kelseyhightower/envconfig"
)

type HttpConfig struct {
	BaseConfig
	Server    HttpServerConfig `envconfig:"HTTP_SERVER"`
	RateLimit RateLimitConfig  `envconfig:"RATE_LIMIT"`
	CORS      CORSConfig       `envconfig:"CORS"`
}

type HttpServerConfig struct {
	Host         string `envconfig:"HOST" default:"0.0.0.0"`
	Port         int    `envconfig:"HTTP_SERVER_PORT" default:"8080"`
	ReadTimeout  int    `envconfig:"READ_TIMEOUT" default:"30"`
	WriteTimeout int    `envconfig:"WRITE_TIMEOUT" default:"30"`
	IdleTimeout  int    `envconfig:"IDLE_TIMEOUT" default:"120"`
}

type RateLimitConfig struct {
	GlobalRequests int `envconfig:"GLOBAL_REQUESTS" default:"1000"`
	GlobalWindow   int `envconfig:"GLOBAL_WINDOW" default:"60"`
	RequestsPerIP  int `envconfig:"REQUESTS_PER_IP" default:"100"`
	WindowSeconds  int `envconfig:"WINDOW_SECONDS" default:"60"`
}

type CORSConfig struct {
	AllowedOrigins   []string `envconfig:"ALLOWED_ORIGINS" default:"*"`
	AllowedMethods   []string `envconfig:"ALLOWED_METHODS" default:"GET,POST,PUT,DELETE,OPTIONS"`
	AllowedHeaders   []string `envconfig:"ALLOWED_HEADERS" default:"Accept,Authorization,Content-Type,X-CSRF-Token"`
	ExposedHeaders   []string `envconfig:"EXPOSED_HEADERS" default:""`
	AllowCredentials bool     `envconfig:"ALLOW_CREDENTIALS" default:"false"`
	MaxAge           int      `envconfig:"MAX_AGE" default:"86400"`
}

func LoadHttp() (*HttpConfig, error) {
	var cfg HttpConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
