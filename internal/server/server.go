package server

import (
	"context"
	"fmt"
	"io"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	"mock-server/internal/server/protocol"
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
const FS_CODE_DIR = "dyn_handle"

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

	s.router = gin.New()

	if cfg.DeployProduction {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

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
	routes := api.Group("routes")

	s.initRoutesApiStatic(routes)
	s.initRoutesApiDynamic(routes)

	// route all query to handle dynamically
	// created user mock endpoints
	s.initNoRoute()
}

// static routes with predefined response
func (s *server) initRoutesApiStatic(routes *gin.RouterGroup) {
	staticRoutesEndpoint := "/static"

	routes.GET(staticRoutesEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all routes static request")
		endpoints, err := database.ListAllStaticEndpointPaths(c)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
	})

	routes.POST(staticRoutesEndpoint, func(c *gin.Context) {
		var staticEndpoint protocol.StaticEndpoint
		if err := c.Bind(&staticEndpoint); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", staticEndpoint.Path).Msg("Received create static request")

		err := database.AddStaticEndpoint(c, staticEndpoint.Path, staticEndpoint.ExpectedResponse)

		switch err {
		case nil:
			zlog.Info().Str("path", staticEndpoint.Path).Msg("Static endpoint created")
			c.JSON(http.StatusOK, "Static endpoint successfully added!")
		case database.ErrDuplicateKey:
			c.JSON(http.StatusConflict, gin.H{"error": "The same endpoint already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	routes.PUT(staticRoutesEndpoint, func(c *gin.Context) {
		var staticEndpoint protocol.StaticEndpoint
		if err := c.Bind(&staticEndpoint); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", staticEndpoint.Path).Msg("Received update static request")

		err := database.UpdateStaticEndpoint(c, staticEndpoint.Path, staticEndpoint.ExpectedResponse)
		switch err {
		case nil:
			zlog.Info().Str("path", staticEndpoint.Path).Msg("Static endpoint updated")
			c.JSON(http.StatusNoContent, "Static endpoint successfully updated!")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Update on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
		case database.ErrBadRouteType:
			zlog.Error().Msg("Update on route with different type")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path has different route type"})
		default:
			zlog.Error().Err(err).Msg("Failed to add static endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	routes.DELETE(staticRoutesEndpoint, func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received delete static request")

		if err := database.RemoveStaticEndpoint(c, path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", path).Msg("Static endpoint removed")
		c.JSON(http.StatusNoContent, "Static endpoint successfully removed!")
	})
}

// dynamic routes (each request involves launching user's code)
func (s *server) initRoutesApiDynamic(routes *gin.RouterGroup) {
	dynamicRoutesEndpoint := "/dynamic"

	routes.GET(dynamicRoutesEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all routes dynamic request")

		endpoints, err := database.ListAllDynamicEndpointPaths(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
	})

	routes.GET(dynamicRoutesEndpoint+"/code", func(c *gin.Context) {
		var dynamicEndpointQuery protocol.DynamicEndpointCodeQuery
		if err := c.Bind(&dynamicEndpointQuery); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", dynamicEndpointQuery.Path).Msg("Received get code dynamic request")

		scriptName, err := database.GetDynamicEndpointScriptName(c, dynamicEndpointQuery.Path)
		switch err {
		case nil:
			zlog.Info().Str("script name", scriptName).Msg("Got script")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Request for unexisting script")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
			return
		default:
			zlog.Error().Err(err).Msg("Failed to query script name")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		code, err := s.fs.Read(FS_CODE_DIR, scriptName)
		if err != nil {
			zlog.Error().Err(err).Str("script name", scriptName).Msg("Failed to read script code")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": util.UnwrapCodeForDynHandle(code)})
	})

	routes.POST(dynamicRoutesEndpoint, func(c *gin.Context) {
		var dynamicEndpoint protocol.DynamicEndpoint
		if err := c.Bind(&dynamicEndpoint); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", dynamicEndpoint.Path).Msg("Received create dynamic request")

		scriptName := util.GenUniqueFilename("py")
		zlog.Info().Str("filename", scriptName).Msg("Generated script name")

		if err := s.fs.Write(FS_CODE_DIR, scriptName, util.WrapCodeForDynHandle(dynamicEndpoint.Code)); err != nil {
			zlog.Error().Err(err).Msg("Failed to write code to file")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		err := database.AddDynamicEndpoint(c, dynamicEndpoint.Path, scriptName)

		switch err {
		case nil:
			zlog.Info().
				Str("path", dynamicEndpoint.Path).
				Str("sciprt name", scriptName).
				Msg("Dynamic endpoint added")
			c.JSON(http.StatusOK, "Dynamic endpoint successfully added")
		case database.ErrDuplicateKey:
			c.JSON(http.StatusConflict, gin.H{"error": "The same endpoint already exists"})
		default:
			zlog.Error().Err(err).Msg("Failed to add dynamic endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	routes.PUT(dynamicRoutesEndpoint, func(c *gin.Context) {
		var dynamicEndpoint protocol.DynamicEndpoint
		if err := c.Bind(&dynamicEndpoint); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", dynamicEndpoint.Path).Msg("Received update dynamic request")

		scriptName, err := database.GetDynamicEndpointScriptName(c, dynamicEndpoint.Path)
		if err == database.ErrNoSuchPath {
			zlog.Error().Msg("Update on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
			return
		}
		if err == database.ErrBadRouteType {
			zlog.Error().Msg("Update on route with different type")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path has different route type"})
			return
		}
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to get script name")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("filename", scriptName).Msg("Script name")

		if err := s.fs.Write(FS_CODE_DIR, scriptName, util.WrapCodeForDynHandle(dynamicEndpoint.Code)); err != nil {
			zlog.Error().Err(err).Msg("Failed to write code to file")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", dynamicEndpoint.Path).Msg("Dynamic endpoint updated")
		c.JSON(http.StatusNoContent, "Dynamic endpoint successfully updated")
	})

	routes.DELETE(dynamicRoutesEndpoint, func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received delete dynamic request")

		if err := database.RemoveDynamicEndpoint(c, path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", path).Msg("Dynamic endpoint removed")
		c.JSON(http.StatusNoContent, "Dynamic endpoint successfully removed")
	})
}

func (s *server) handleStaticRouteRequest(c *gin.Context, route database.Route) {
	c.JSON(http.StatusOK, route.Response)
}

func (s *server) handleDynamicRouteRequest(c *gin.Context, route database.Route) {

	worker, err := coderun.WorkerWatcher.BorrowWorker()
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to borrow worker")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer worker.Return()

	args, err := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := worker.RunScript(FS_CODE_DIR, route.ScriptName, coderun.NewDynHandleArgs(args))
	switch err {
	case nil:
		c.JSON(http.StatusOK, string(output))
	case coderun.ErrCodeRunFailed:
		zlog.Warn().Str("output", string(output)).Msg("Failed to run script")
		c.JSON(http.StatusBadRequest, gin.H{"error": string(output)})
	default:
		zlog.Error().Str("output", string(output)).Msg("Worker failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": string(output)})
	}
}

func (s *server) initNoRoute() {
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		zlog.Info().Str("path", path).Msg("Received path")

		route, err := database.GetRoute(c, path)
		if err == database.ErrNoSuchPath {
			zlog.Info().Str("path", path).Msg("No such path")
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("no such path: %s", path)})
			return
		}
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to find path")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		switch route.Type {
		case database.STATIC_ENDPOINT_TYPE:
			s.handleStaticRouteRequest(c, route)

		case database.DYNAMIC_ENDPOINT_TYPE:
			s.handleDynamicRouteRequest(c, route)

		default:
			zlog.Error().Msg(fmt.Sprintf("Can't resolve route type: %s", route.Type))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Can't resolve route type"})
			return
		}
	})
}
