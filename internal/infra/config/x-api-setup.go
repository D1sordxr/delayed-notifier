package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

const basicApiConfigPath = "./configs/api/prod.yaml"

type ApiConfig struct {
	Cache   Redis      `yaml:"cache"`
	Storage Postgres   `yaml:"storage"`
	Server  HTTPServer `yaml:"server"`
}

func NewApiConfig() *ApiConfig {
	var cfg ApiConfig

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = basicApiConfigPath
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
