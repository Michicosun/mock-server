package configs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	zlog "github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Logs    LogConfig         `yaml:"logs"`
	Pool    PoolConfig        `yaml:"pool"`
	Brokers BrokerConnections `yaml:"brokers"`
}

var config ServiceConfig

func LoadConfig(cfg_path string) {
	path, err := os.Getwd()
	if err != nil {
		zlog.Err(err)
		panic(err)
	}

	filename, _ := filepath.Abs(filepath.Join(path, cfg_path))
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		zlog.Err(err).Msg("failed to read config file")
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		zlog.Err(err).Msg("Unmarshal config failed")
		panic(err)
	}

	s, _ := json.MarshalIndent(config, "", "\t")
	fmt.Println("Config", string(s))
}
