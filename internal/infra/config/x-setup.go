package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

const basicConfigPath = "./configs/app.yaml"

var configPath string

type Config struct {
	Cache   Redis      `yaml:"cache"`
	Storage Postgres   `yaml:"storage"`
	Broker  RabbitMQ   `yaml:"broker"`
	Server  HTTPServer `yaml:"server"`
}

func NewConfig() *Config {
	var cfg Config

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = basicConfigPath
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
