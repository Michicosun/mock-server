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

	secret_id, _ := brokers.SecretBox.SetSecret(&brokers.RabbitMQSecret{
		Username: "guest",
		Password: "guest",
		Host:     "localhost",
		Port:     5672,
		Queue:    "test-mock-queue",
	})

	conn := brokers.NewRabbitMQConnection(secret_id)

	conn.Write([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Submit()

	<-time.After(1 * time.Second)

	conn.Read().Submit()

	zlog.Info().Msg("start reading")
	<-time.After(30 * time.Second)

	cancel()

	brokers.BrokerPool.Wait()
}
