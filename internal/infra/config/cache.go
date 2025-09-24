package config

type Redis struct {
	ClientAddress string `yaml:"client_address"`
	Password      string `yaml:"password"`
}
