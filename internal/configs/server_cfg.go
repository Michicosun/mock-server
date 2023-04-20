package configs

import "time"

type ServerConfig struct {
	Addr            string        `yaml:"addr"`
	Port            string        `yaml:"port"`
	AcceptTimeout   time.Duration `yaml:"accept_timeout"`
	ResponseTimeout time.Duration `yaml:"response_timeout"`
}

func GetServerConfig() *ServerConfig {
	return &config.Server
}
