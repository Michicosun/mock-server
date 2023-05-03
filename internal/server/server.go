package server

import (
	"context"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	requesttypes "mock-server/internal/server/request_types"
	"mock-server/internal/util"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	reuseport "github.com/kavu/go_reuseport"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

var Server = &server{}

type server struct {
	server_instance *http.Server
	router          *gin.Engine
	fs              *util.FileStorage
}

func (s *server) Init(cfg *configs.ServerConfig) {
	{
		fs, err := util.NewFileStorageDriver("dyn_handlers")
		if err != nil {
			panic(err)
		}
		s.fs = fs
	}

	s.router = gin.New()

	s.router.Use(logger.GinLogger()) // use custom logger (zerolog)
	s.router.Use(gin.Recovery())     // recovery from all panics

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
	s.initRoutesApi(api)

	// route all query to handle dynamically
	// created user mock endpoints
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		zlog.Info().Str("path", path).Msg("Received path")

		expectedResponse, err := database.GetStaticEndpointResponse(path)
		if err == nil {
			c.JSON(http.StatusOK, expectedResponse)
			return
		}
		if err != database.ErrNoSuchPath {
			zlog.Error().Err(err).Msg("Failed to get static endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		scriptName, err := database.GetDynamicEndpointScriptName(path)
		if err == nil {
			worker, err := coderun.WorkerWatcher.BorrowWorker()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			type ComplexArgs struct {
				A string   `json:"A"`
				B int      `json:"B"`
				C []string `json:"C"`
			}

			if err != nil {
				zlog.Error().Err(err).Str("path", path).Msg("Failed to get script name")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			output, err := worker.RunScript("dyn_handle", scriptName, ComplexArgs{
				A: "sample_A",
				B: 42,
				C: []string{"a", "b", "c"},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, string(output))
			return
		}
		if err != database.ErrNoSuchPath {
			zlog.Error().Err(err).Msg("Failed to get dynamic endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", path).Msg("No such path")
		c.JSON(http.StatusBadRequest, gin.H{"error": database.ErrNoSuchPath.Error()})
	})
}

func (s *server) initRoutesApi(apiGroup *gin.RouterGroup) {
	routes := apiGroup.Group("routes")

	// static routes
	{
		staticRoutesEndpoint := "/static"

		routes.GET(staticRoutesEndpoint, func(c *gin.Context) {
			zlog.Info().Msg("Get all routes static request")
			endpoints, err := database.ListAllStaticEndpointPaths()

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": errors.Wrap(err, "internal error").Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
		})

		routes.POST(staticRoutesEndpoint, func(c *gin.Context) {
			var staticEndpoint database.StaticEndpoint
			if err := c.Bind(&staticEndpoint); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": errors.Wrap(err, "bad request").Error()})
				return
			}

			zlog.Info().Str("path", staticEndpoint.Path).Msg("Received create static request")

			_, err := database.GetStaticEndpointResponse(staticEndpoint.Path)

			if err != nil && err != database.ErrNoSuchPath {
				c.JSON(http.StatusInternalServerError, gin.H{"error": errors.Wrap(err, "internal error").Error()})
				return
			}

			if err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "The same static endpoint already exists"})
				return
			}

			if err := database.AddStaticEndpoint(staticEndpoint); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": errors.Wrap(err, "internal error").Error()})
				return
			}

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

			if err := database.RemoveStaticEndpoint(path); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": errors.Wrap(err, "internal error").Error()})
				return
			}

			zlog.Info().Str("path", path).Msg("Static endpoint removed")
			c.JSON(http.StatusNoContent, "Static endpoint successfully removed!")
		})
	}

	// dynamic routes (each request involves launching user's code)
	{
		dynamicRoutesEndpoint := "/dynamic"

		routes.GET(dynamicRoutesEndpoint, func(c *gin.Context) {
			zlog.Info().Msg("Get all routes dynamic request")

			endpoints, err := database.ListAllDynamicEndpointPaths()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
		})

		routes.POST(dynamicRoutesEndpoint, func(c *gin.Context) {
			var dynamicEndpoint requesttypes.DynamicEndpoint
			if err := c.Bind(&dynamicEndpoint); err != nil {
				zlog.Error().Err(err).Msg("Failed to bind request")
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			zlog.Info().Str("path", dynamicEndpoint.Path).Msg("Received create dynamic request")

			scriptName := util.GenUniqueFilename("py")
			zlog.Info().Str("filename", scriptName).Msg("Generated script name")

			if err := s.fs.Write("", scriptName, []byte(dynamicEndpoint.Code)); err != nil {
				zlog.Error().Err(err).Msg("Failed to write code to file")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			database.AddDynamicEndpoint(database.DynamicEndpoint{
				Path:       dynamicEndpoint.Path,
				ScriptName: scriptName,
			})
			zlog.Info().
				Str("path", dynamicEndpoint.Path).
				Str("sciprt name", scriptName).
				Msg("Dynamic endpoint added")

			c.JSON(http.StatusOK, "Dynamic endpoint successfully added")
		})

		// TODO add PUT for script updating

		routes.DELETE(dynamicRoutesEndpoint, func(c *gin.Context) {
			path := c.Query("path")
			if path != "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
				return
			}

			zlog.Info().Str("path", path).Msg("Received delete dynamic request")

			database.RemoveDynamicEndpoint(path)

			zlog.Info().Str("path", path).Msg("Dynamic endpoint removed")
			c.JSON(http.StatusNoContent, "Dynamic endpoint successfully removed")
		})
	}
}
