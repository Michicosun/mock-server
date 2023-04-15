package server

import (
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	requesttypes "mock-server/internal/server/request_types"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

var Server = &server{}

type server struct {
	constructor sync.Once

	server_instance *http.Server
	router          *gin.Engine
	db              database.Database
}

func (s *server) Init(cfg *configs.ServerConfig, database database.Database) {
	s.constructor.Do(func() {
		s.db = database

		s.router = gin.New()

		s.router.Use(Logger())       // use custom logger (zerolog)
		s.router.Use(gin.Recovery()) // recovery from all panics

		s.initMainRoutes()

		s.server_instance = &http.Server{
			Addr:         fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port),
			Handler:      s.router,
			ReadTimeout:  cfg.AcceptTimeout,
			WriteTimeout: cfg.ResponseTimeout,
		}
	})
}

func (s *server) Start() {
	s.server_instance.ListenAndServe()
}

func (s *server) initMainRoutes() {
	// just ping
	s.router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Ping yourself, I have another work!\n")
	})

	// static routes
	static := s.router.Group("static")
	{
		static.POST("/add", func(c *gin.Context) {
			var staticEndpoint requesttypes.StaticEndpoint
			if err := c.Bind(&staticEndpoint); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": errors.Wrap(err, "bad request").Error()})
				return
			}

			if s.db.PeekStaticEndpoint(staticEndpoint.Path) {
				c.JSON(http.StatusConflict, "The same static endpoint already exists")
				return
			}

			s.db.AddStaticEndpoint(staticEndpoint.Path, []byte(staticEndpoint.ExpectedResponse))
			c.JSON(http.StatusOK, "Static endpoint successfully added!")
		})

		static.DELETE("/remove", func(c *gin.Context) {
			var path requesttypes.StaticEndpointPath
			if err := c.Bind(&path); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": errors.Wrap(err, "bad request").Error()})
				return
			}

			s.db.RemoveStaticEndpoint(path.Path)
			c.String(http.StatusOK, "Static endpoint successfully removed!")
		})
	}
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		zlog.Info().Str("path", path).Msg("got path")

		expected_response, err := s.db.GetStaticEndpointResponse(path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errors.Wrap(err, "bad request").Error()})
			return
		}

		c.JSON(http.StatusOK, expected_response)
	})
}
