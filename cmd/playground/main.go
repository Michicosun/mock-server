package main

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun/docker-provider"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/util"
	"time"

	zlog "github.com/rs/zerolog/log"
)

func play_brokers(ctx context.Context, cancel context.CancelFunc) {
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

	<-time.After(10 * time.Second)

	cancel()

	brokers.BrokerPool.Stop()

	for x := range brokers.BrokerPool.Errors() {
		fmt.Println(x)
	}
}

func play_docker(ctx context.Context, cancel context.CancelFunc) {
	provider, err := docker.NewDockerProvider(ctx, &configs.GetCoderunConfig().DockerContainerResources)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create provider")
		return
	}

	err = provider.BuildWorkerImage()
	if err != nil {
		zlog.Error().Err(err).Msg("cannot build image")
		return
	}

	id, err := provider.CreateWorkerContainer("8095")
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create container")
		return
	}

	err = provider.StartWorkerContainer(id)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create container")
		return
	}

	time.Sleep(time.Second * 100)

	err = provider.RemoveWorkerContainer(id, true)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot remove container")
		return
	}
	provider.Close()
}

func play_file_storage(ctx context.Context, cancel context.CancelFunc) {
	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create filestorage")
		return
	}
	err = fs.Write("mappers", "a", []byte("Hello, world!"))
	if err != nil {
		zlog.Error().Err(err).Msg("write failed")
		return
	}
	s, err := fs.Read("mappers", "a")
	if err != nil {
		zlog.Error().Err(err).Msg("read failed")
		return
	}
	zlog.Info().Str("text", s).Msg("read file successfuly")
}

func main() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	// create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zlog.Info().Msg("starting...")

	// play_brokers(ctx, cancel)
	play_docker(ctx, cancel)
	// play_file_storage(ctx, cancel)
}