package test_lib

import (
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	"time"
)

func InitTest() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	// init database
	database.DB.Drop()

	// wait other containers
	<-time.After(5 * time.Second)
}
