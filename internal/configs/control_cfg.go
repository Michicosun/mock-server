package configs

type ComponentsConfig struct {
	Brokers bool `yaml:"brokers"`
	Coderun bool `yaml:"coderun"`
	Server  bool `yaml:"server"`
}

func GetComponentsConfig() *ComponentsConfig {
	return &config.Components
}
