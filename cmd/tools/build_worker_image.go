package main

import (
	"context"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"os"

	zlog "github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Init(&configs.LogConfig{
		Level:                 0,
		ConsoleLoggingEnabled: true,
		FileLoggingEnabled:    false,
	})

	if _, set := os.LookupEnv("SKIP_REBUILD_CONTAINER"); set {
		zlog.Warn().Msg("Manually skip rebuild container")
		return
	}

	if err := coderun.WorkerWatcher.Init(ctx, &configs.CoderunConfig{
		WorkerCnt: 0,
	}); err != nil {
		panic(err)
	}
}
