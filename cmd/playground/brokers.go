package main

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"os"
	"time"

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

	// create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zlog.Info().Msg("starting...")

	// broker example
	brokers.BrokerPool.Init(ctx, configs.GetPoolConfig())
	brokers.BrokerPool.Start()

	brokers.BrokerPool.NewRabbitMQWriteTask("test-mock-queue").Write([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	})

	<-time.After(1 * time.Second)

	id := brokers.BrokerPool.NewRabbitMQReadTask("test-mock-queue").Read()

	zlog.Info().Msg("start reading")
	<-time.After(5 * time.Second)

	brokers.BrokerPool.StopEventually(id)

	<-time.After(20 * time.Second)

	cancel()

	brokers.BrokerPool.Wait()
}
