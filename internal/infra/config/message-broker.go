package config

import (
	"fmt"
	"strings"
)

type RabbitMQ struct {
	Host     string `yaml:"host" env:"RABBITMQ_HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"RABBITMQ_PORT" env-default:"5672"`
	Username string `yaml:"username" env:"RABBITMQ_USERNAME" env-default:"admin"`
	Password string `yaml:"password" env:"RABBITMQ_PASSWORD" env-default:"delayed-notifier-password"`
	VHost    string `yaml:"vhost" env:"RABBITMQ_VHOST" env-default:"/"`
}

func (c *RabbitMQ) GetConnectionString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		strings.TrimPrefix(c.VHost, "/"))
}
