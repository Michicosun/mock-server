package server

import (
	"context"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/util"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	reuseport "github.com/kavu/go_reuseport"
	zlog "github.com/rs/zerolog/log"
)

var Server = &server{}

const FS_ROOT_DIR = "coderun"
const FS_DYN_HANDLE_DIR = "dyn_handle"
const FS_ESB_DIR = "mapper"

type server struct {
	server_instance *http.Server
	router          *gin.Engine
	fs              *util.FileStorage
}

func (s *server) Init(cfg *configs.ServerConfig) {
	{
		fs, err := util.NewFileStorageDriver(FS_ROOT_DIR)
		if err != nil {
			panic(err)
		}
		s.fs = fs
	}

	if cfg.DeployProduction {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s.router = gin.New()

	s.router.Use(logger.GinLogger()) // use custom logger (zerolog)
	s.router.Use(gin.Recovery())     // recovery from all panics
	s.router.Use(cors.Default())     // needs when routing development-mode frontend app

	s.initMainRoutes()

	s.server_instance = &http.Server{
		Addr:         cfg.Addr,
		Handler:      s.router,
		ReadTimeout:  cfg.AcceptTimeout,
		WriteTimeout: cfg.ResponseTimeout,
	}
}

func (s *server) Start() {
	ch := make(chan interface{})

	go func() {
		zlog.Info().Msg("starting server")

		ch <- struct{}{}

		ln, err := reuseport.Listen("tcp", s.server_instance.Addr)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to create listener")
			panic(err)
		}

		zlog.Info().
			Str("addr", ln.Addr().String()).
			Msg("Server listens")

		if err := s.server_instance.Serve(ln); err != nil && err != http.ErrServerClosed {
			zlog.Error().Err(err).Msg("failure while server working")
			panic(err)
		}
	}()

	<-ch

	// wait until server start listening
	time.Sleep(1 * time.Second)
}

func (s *server) Stop() {
	zlog.Info().Msg("stopping server with timeout 5 seconds")
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server_instance.Shutdown(timeout); err != nil {
		zlog.Fatal().Err(err).Msg("Server forced to shutdown")
	}
}

func (s *server) initMainRoutes() {
	api := s.router.Group("api")

	// just ping
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, "Ping yourself, I have another work!")
		})
	}

	// init routes (static, proxy, dynamic)
	routesApi := api.Group("routes")

	s.initRoutesApiStatic(routesApi)
	s.initRoutesApiDynamic(routesApi)
	s.initRoutesApiProxy(routesApi)

	// route all query to handle dynamically
	// created user mock endpoints
	s.initNoRoute()

	// init brokers (message pools, task scheduling and ESB)
	brokersApi := api.Group("brokers")

	s.initBrokersApiPool(brokersApi)
	s.initBrokersApiEsb(brokersApi)
}
