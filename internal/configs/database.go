package configs

type DatabaseConfig struct {
	InMemory  bool `yaml:"inmemory"`
	CacheSize int  `yaml:"cache_size"`
}

func GetDatabaseConfig() *DatabaseConfig {
	return &config.Database
}
