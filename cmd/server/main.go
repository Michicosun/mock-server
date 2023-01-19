package main

import (
	"context"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/logger"

	zlog "github.com/rs/zerolog/log"
)

func main() {
	// load config
	configs.LoadConfig()

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
