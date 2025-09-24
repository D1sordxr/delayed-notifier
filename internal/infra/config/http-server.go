package config

import (
	"time"
)

type HTTPServer struct {
	Address      string        `yaml:"address" env:"HTTP_ADDRESS" env-default:"localhost"`
	Port         string        `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
	Timeout      time.Duration `yaml:"timeout" env:"HTTP_TIMEOUT" env-default:"4s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	CORS         bool          `yaml:"cors" env:"HTTP_CORS"`
	AllowOrigins []string      `yaml:"allow_origins" env:"HTTP_ALLOWED_ORIGINS"`
}
