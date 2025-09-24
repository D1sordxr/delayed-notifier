package config

import "fmt"

type Postgres struct {
	Host       string `yaml:"host" env:"PG_HOST"`
	Port       int    `yaml:"port" env:"PG_PORT"`
	Database   string `yaml:"database" env:"PG_DATABASE"`
	User       string `yaml:"user" env:"PG_USER"`
	Password   string `yaml:"password" env:"PG_PASSWORD"`
	Migrations bool   `yaml:"migrations" env:"PG_MIGRATIONS"`
	SSLMode    string `yaml:"ssl_mode" env:"PG_SSLMODE" env-default:"disable"`
}

func (p *Postgres) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		p.Host, p.User, p.Password, p.Database, p.Port, p.SSLMode,
	)
}
