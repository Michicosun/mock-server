package main

import (
	"context"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Init(&configs.LogConfig{
		Level:                 0,
		ConsoleLoggingEnabled: true,
		FileLoggingEnabled:    false,
	})

	coderun.WorkerWatcher.Init(ctx, &configs.CoderunConfig{
		WorkerCnt: 0,
	})
}
