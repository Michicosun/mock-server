package server

import (
	"context"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	requesttypes "mock-server/internal/server/request_types"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

var Server = &server{}

type server struct {
	server_instance *http.Server
	router          *gin.Engine
	db              database.Database
}

func (s *server) Init(cfg *configs.ServerConfig) {
	s.db = database.DB

	s.router = gin.New()

	s.router.Use(logger.GinLogger()) // use custom logger (zerolog)
	s.router.Use(gin.Recovery())     // recovery from all panics

	s.initMainRoutes()

	s.server_instance = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.AcceptTimeout,
		WriteTimeout: cfg.ResponseTimeout,
	}
}

func (s *server) Start() {
	go func() {
		zlog.Info().Msg("starting server")

		lc := net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				err := c.Control(func(fd uintptr) {
					opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				})
				if err != nil {
					return err
				}
				return opErr
			},
		}

		ln, err := lc.Listen(context.Background(), "tcp", s.server_instance.Addr)
		if err != nil {
			zlog.Fatal().Err(err).Msg("listen configuration failure")
		}

		if err := s.server_instance.Serve(ln); err != nil && err != http.ErrServerClosed {
			zlog.Error().Err(err).Msg("failure while server working")
		}
	}()
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
			c.JSON(http.StatusOK, "Ping yourself, I have another work!\n")
		})
	}

	// init routes (static, proxy, dynamic)
	s.initRoutesApi(api)

	// route all query to handle dynamically
	// created user mock endpoints
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		zlog.Info().Str("path", path).Msg("Received path")

		expected_response, err := s.db.GetStaticEndpointResponse(path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, expected_response)
	})
}

func (s *server) initRoutesApi(apiGroup *gin.RouterGroup) {
	routes := apiGroup.Group("routes")

	// static routes
	{
		staticRoutesEndpoint := "/static"

		routes.GET(staticRoutesEndpoint, func(c *gin.Context) {
			endpoints := s.db.ListAllStaticEndpoints()

			c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
		})

		routes.POST(staticRoutesEndpoint, func(c *gin.Context) {
			var staticEndpoint requesttypes.StaticEndpoint
			if err := c.Bind(&staticEndpoint); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": errors.Wrap(err, "bad request").Error()})
				return
			}

			zlog.Info().Str("path", staticEndpoint.Path).Msg("Received create static request")

			if s.db.HasStaticEndpoint(staticEndpoint.Path) {
				c.JSON(http.StatusConflict, "The same static endpoint already exists")
				return
			}

			s.db.AddStaticEndpoint(staticEndpoint.Path, staticEndpoint.ExpectedResponse)

			zlog.Info().Str("path", staticEndpoint.Path).Msg("Static endpoint created")
			c.JSON(http.StatusOK, "Static endpoint successfully added!")
		})

		routes.DELETE(staticRoutesEndpoint, func(c *gin.Context) {
			path := c.Query("path")
			if path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
				return
			}

			zlog.Info().Str("path", path).Msg("Received delete static request")

			s.db.RemoveStaticEndpoint(path)

			zlog.Info().Str("path", path).Msg("Static endpoint removed")
			c.String(http.StatusOK, "Static endpoint successfully removed!")
		})
	}
}
