package configs

import (
	"encoding/json"
	"fmt"
	"mock-server/internal/util"
	"os"
	"path/filepath"

	zlog "github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "/configs/config.yaml"

type ServiceConfig struct {
	Logs    LogConfig     `yaml:"logs"`
	Brokers BrokersConfig `yaml:"brokers"`
	Coderun CoderunConfig `yaml:"coderun"`
	Server  ServerConfig  `yaml:"server"`
}

var config ServiceConfig

func LoadConfig() {
	cfg_path := os.Getenv("CONFIG_PATH")
	if cfg_path == "" {
		cfg_path = DefaultConfigPath
	}

	path, err := util.GetProjectRoot()
	if err != nil {
		zlog.Err(err).Msg("undefined project root")
		panic(err)
	}

	full_cfg_path, err := filepath.Abs(filepath.Join(path, cfg_path))
	if err != nil {
		zlog.Err(err).Msg("failed to create full config path")
		panic(err)
	}
	cfg, err := os.ReadFile(full_cfg_path)

	if err != nil {
		zlog.Err(err).Msg("failed to read config file")
		panic(err)
	}

	if err = yaml.Unmarshal(cfg, &config); err != nil {
		zlog.Err(err).Msg("Unmarshal config failed")
		panic(err)
	}

	s, err := json.MarshalIndent(config, "", "\t")
	if err == nil {
		fmt.Println("Config", string(s))
	}
}
