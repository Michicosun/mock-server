package configs

type ContainerResources struct {
	CPULimit    float64 `yaml:"cpu_limit"`
	MemoryLimit float64 `yaml:"memory_limit"`
}

type CoderunConfig struct {
	DockerContainerResources ContainerResources `yaml:"docker_container"`
	WorkerCnt                int                `yaml:"worker_cnt"`
}

func GetCoderunConfig() *CoderunConfig {
	return &config.Coderun
}
