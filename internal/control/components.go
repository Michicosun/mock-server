package control

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	"mock-server/internal/server"
	"os/signal"
	"syscall"

	zlog "github.com/rs/zerolog/log"
)

var Components = &componentsManager{}

type componentsManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *configs.ComponentsConfig
}

func (c *componentsManager) Start() {
	// make root context
	c.ctx, c.cancel = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// load config
	configs.LoadConfig()
	c.cfg = configs.GetComponentsConfig()

	// init logger
	logger.Init(configs.GetLogConfig())
	zlog.Info().Msg("starting...")

	err := database.InitDB(c.ctx, configs.GetDatabaseConfig())
	if err != nil {
		panic(err)
	}

	// init pool registry
	brokers.MPRegistry.Init()

	// init esb
	brokers.Esb.Init()

	// start server
	if c.cfg.Server {
		server.Server.Init(configs.GetServerConfig())
		server.Server.Start()
	}

	// start broker tasks scheduler
	if c.cfg.Brokers {
		brokers.MPTaskScheduler.Init(c.ctx, configs.GetMPTaskSchedulerConfig())
		brokers.MPTaskScheduler.Start()
	}

	// start coderun
	if c.cfg.Coderun {
		err := coderun.WorkerWatcher.Init(c.ctx, configs.GetCoderunConfig())
		if err != nil {
			panic(fmt.Errorf("coderun expected to init but failed: %s", err.Error()))
		}
	}
}

func (c *componentsManager) Wait() {
	// wait for os interrupt signal
	<-c.ctx.Done()
}

func (c *componentsManager) Stop() {
	// send stop signal
	c.cancel()

	// wait for server shutdown
	if c.cfg.Server {
		server.Server.Stop()
	}

	// wait for brokers shutdown
	if c.cfg.Brokers {
		brokers.MPTaskScheduler.Stop()
	}

	// wait for coderun shutdown
	if c.cfg.Coderun {
		coderun.WorkerWatcher.Stop()
	}

	database.Disconnect(c.ctx)
}
