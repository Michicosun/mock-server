package configs

import "time"

type ContainerConfig struct {
	NeedRebuild   bool    `yaml:"need_rebuild"`
	CPULimit      float64 `yaml:"cpu_limit"`
	MemoryLimitMB float64 `yaml:"memory_limit_mb"`
}

type WorkerConfig struct {
	ContainerConfig ContainerConfig `yaml:"container"`
	HandleTimeout   time.Duration   `yaml:"handle_timeout"`
}

type CoderunConfig struct {
	WorkerConfig WorkerConfig `yaml:"worker"`
	WorkerCnt    int          `yaml:"worker_cnt"`
}

func GetCoderunConfig() *CoderunConfig {
	return config.Coderun
}
