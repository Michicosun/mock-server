package configs

type DatabaseConfig struct {
	InMemory bool `yaml:"inmemory"`
}

func GetDatabaseConfig() *DatabaseConfig {
	return &config.Database
}
