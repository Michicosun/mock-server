package main

import (
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/server"
)

func main() {
	// load config
	configs.LoadConfig()

	logger.Init(configs.GetLogConfig())

	server.Server.Init(configs.GetServerConfig())
	server.Server.Start() // blocks until manual interruptions
}
