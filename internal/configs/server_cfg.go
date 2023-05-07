package configs

import "time"

type ServerConfig struct {
	Addr             string        `yaml:"addr"`
	AcceptTimeout    time.Duration `yaml:"accept_timeout"`
	ResponseTimeout  time.Duration `yaml:"response_timeout"`
	DeployProduction bool          `yaml:"deploy_production"`
}

func GetServerConfig() *ServerConfig {
	return config.Server
}
