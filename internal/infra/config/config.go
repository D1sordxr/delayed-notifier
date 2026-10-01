package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/D1sordxr/packages/httpserver"
	"github.com/D1sordxr/packages/postgres"
	"github.com/D1sordxr/packages/rabbitmq"
	"github.com/D1sordxr/packages/redis"
	"github.com/ilyakaznacheev/cleanenv"
)

type Postgres struct {
	postgres.Config `yaml:",inline"`
	Migrations      bool `yaml:"migrations"`
}

type Cache struct {
	redis.Config `yaml:",inline"`
	TTL          time.Duration `yaml:"ttl" env-default:"10m"`
	// WriteBuffer and WriteWorkers size the async cache writer.
	WriteBuffer  int `yaml:"write_buffer" env-default:"1024"`
	WriteWorkers int `yaml:"write_workers" env-default:"2"`
}

type HTTPServer struct {
	httpserver.Config `yaml:",inline"`
	CORS              bool     `yaml:"cors"`
	AllowOrigins      []string `yaml:"allow_origins"`
}

type Scheduler struct {
	Interval   time.Duration `yaml:"interval" env-default:"1s"`
	Lookahead  time.Duration `yaml:"lookahead" env-default:"1s"`
	BatchSize  int32         `yaml:"batch_size" env-default:"100"`
	StaleAfter time.Duration `yaml:"stale_after" env-default:"5m"`
}

type Dispatcher struct {
	Prefetch    int           `yaml:"prefetch" env-default:"10"`
	MaxAttempts int16         `yaml:"max_attempts" env-default:"3"`
	RetryDelay  time.Duration `yaml:"retry_delay" env-default:"30s"`
}

type ApiConfig struct {
	LogLevel string     `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	Storage  Postgres   `yaml:"storage"`
	Cache    Cache      `yaml:"cache"`
	Server   HTTPServer `yaml:"server"`
}

type WorkerConfig struct {
	LogLevel   string          `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	Storage    Postgres        `yaml:"storage"`
	Cache      Cache           `yaml:"cache"`
	Broker     rabbitmq.Config `yaml:"broker"`
	Scheduler  Scheduler       `yaml:"scheduler"`
	Dispatcher Dispatcher      `yaml:"dispatcher"`
}

const (
	basicApiConfigPath    = "./configs/api/prod.yaml"
	basicWorkerConfigPath = "./configs/worker/prod.yaml"
)

func NewApiConfig() *ApiConfig {
	var cfg ApiConfig
	mustRead(basicApiConfigPath, &cfg)
	return &cfg
}

func NewWorkerConfig() *WorkerConfig {
	var cfg WorkerConfig
	mustRead(basicWorkerConfigPath, &cfg)

	if err := cfg.validate(); err != nil {
		panic("invalid config: " + err.Error())
	}

	return &cfg
}

func (c *WorkerConfig) validate() error {
	s, d := c.Scheduler, c.Dispatcher

	if s.Interval <= 0 || s.BatchSize <= 0 || s.Lookahead < 0 {
		return errors.New("scheduler: interval and batch_size must be positive, lookahead non-negative")
	}
	// A notification waiting in the wait or retry queue is still pending; it
	// must not look stale before it comes back from there.
	if minStale := s.Lookahead + d.RetryDelay + s.Interval; s.StaleAfter <= minStale {
		return fmt.Errorf("scheduler: stale_after must exceed lookahead + retry_delay + interval (%s)", minStale)
	}

	return nil
}

func mustRead(defaultPath string, cfg any) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = defaultPath
	}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}
}
