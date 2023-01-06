package configs

import "time"

type PoolConfig struct {
	R_workers     uint32        `yaml:"r_workers"`
	W_workers     uint32        `yaml:"w_workers"`
	Read_timeout  time.Duration `yaml:"read_timeout"`
	Write_timeout time.Duration `yaml:"write_timeout"`
	Disable_task  time.Duration `yaml:"disable_task"`
}

func GetPoolConfig() *PoolConfig {
	return &config.Pool
}
