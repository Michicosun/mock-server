package configs

type LogConfig struct {
	Level                 uint8  `yaml:"level"`
	ConsoleLoggingEnabled bool   `yaml:"consoleLoggingEnabled"`
	FileLoggingEnabled    bool   `yaml:"fileLoggingEnabled"`
	Directory             string `yaml:"directory"`
	Filename              string `yaml:"filename"`
	MaxSize               int    `yaml:"maxSize"`
	MaxBackups            int    `yaml:"maxBackups"`
	MaxAge                int    `yaml:"maxAge"`
}

func GetLogConfig() *LogConfig {
	return &config.Logs
}
