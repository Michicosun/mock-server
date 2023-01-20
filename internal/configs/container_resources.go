package configs

import "time"

type ContainerResources struct {
	CPULimit    float64 `yaml:"cpu_limit"`
	MemoryLimit float64 `yaml:"memory_limit"`
}

type WorkerConfig struct {
	Resources     ContainerResources `yaml:"resources"`
	HandleTimeout time.Duration      `yaml:"handle_timeout"`
}

type CoderunConfig struct {
	WorkerConfig WorkerConfig `yaml:"worker"`
	WorkerCnt    int          `yaml:"worker_cnt"`
}

func GetCoderunConfig() *CoderunConfig {
	return &config.Coderun
}
