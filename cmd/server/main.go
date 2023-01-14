package main

import (
	"context"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"os"

	zlog "github.com/rs/zerolog/log"
)

const DefaultConfigPath = "/configs/config.yaml"

func main() {
	config_path := os.Getenv("CONFIG_PATH")
	if config_path == "" {
		config_path = DefaultConfigPath
	}

	// load config
	configs.LoadConfig(config_path)

	// init logger
	logger.Init(configs.GetLogConfig())

	zlog.Info().Msg("starting...")

	// create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// complete database prepare scripts
	// TODO
	////////////////////////////////////

	// startup broker pool
	brokers.BrokerPool.Init(ctx, configs.GetPoolConfig())
	brokers.BrokerPool.Start()

	// configure router
	// startup httpserver
	// TODO
	////////////////////////////////////

	cancel()

	brokers.BrokerPool.Stop()
}
