package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

const basicWorkerConfigPath = "./configs/worker/prod.yaml"

type WorkerConfig struct {
	Cache   Redis    `yaml:"cache"`
	Storage Postgres `yaml:"storage"`
	Broker  RabbitMQ `yaml:"broker"`
}

func NewWorkerConfig() *WorkerConfig {
	var cfg WorkerConfig

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = basicWorkerConfigPath
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
