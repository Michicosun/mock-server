package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"mock-server/internal/logger"
	"mock-server/internal/server/protocol"
	"mock-server/internal/util"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	reuseport "github.com/kavu/go_reuseport"
	"github.com/rs/zerolog"
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
	s.initBrokersApiScheduler(brokersApi)
	s.initBrokersApiEsb(brokersApi)
}

// static routes with predefined response
func (s *server) initRoutesApiStatic(routes *gin.RouterGroup) {
	staticRoutesEndpoint := "/static"

	routes.GET(staticRoutesEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all routes static request")
		endpoints, err := database.ListAllStaticEndpointPaths(c)

		if err != nil {
			zlog.Error().Err(err).Msg("Failed to list all static endpoints paths")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
	})

	routes.GET(staticRoutesEndpoint+"/expected_response", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received get proxy url proxy request")

		expectedResponse, err := database.GetStaticEndpointResponse(c, path)
		switch err {
		case nil:
			zlog.Info().Str("expected response ", expectedResponse).Msg("Got url")
		case database.ErrNoSuchPath, database.ErrBadRouteType:
			zlog.Error().Msg("Request for unexisting static route")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
			return
		default:
			zlog.Error().Err(err).Msg("Failed to query expected response")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, expectedResponse)
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
			zlog.Error().Str("path", staticEndpoint.Path).Msg("Endpoint with this path already exists")
			c.JSON(http.StatusConflict, gin.H{"error": "The same endpoint already exists"})
		default:
			zlog.Error().Err(err).Msg("Failed to add static endpoint")
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

		err := database.RemoveStaticEndpoint(c, path)

		switch err {
		case nil:
			zlog.Info().Str("path", path).Msg("Static endpoint removed")
			c.JSON(http.StatusNoContent, "Static endpoint successfully removed!")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Delete on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
		default:
			zlog.Error().Err(err).Msg("Failed to remove static endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})
}

// proxy routes (proxies to predefined url)
func (s *server) initRoutesApiProxy(routes *gin.RouterGroup) {
	proxyRoutesEndpoint := "/proxy"

	routes.GET(proxyRoutesEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all routes proxy request")
		endpoints, err := database.ListAllProxyEndpointPaths(c)

		if err != nil {
			zlog.Error().Err(err).Msg("Failed to list all proxy endpoints paths")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
	})

	routes.GET(proxyRoutesEndpoint+"/proxy_url", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received get proxy url proxy request")

		proxyUrl, err := database.GetProxyEndpointProxyUrl(c, path)
		switch err {
		case nil:
			zlog.Info().Str("proxy url ", proxyUrl).Msg("Got url")
		case database.ErrNoSuchPath, database.ErrBadRouteType:
			zlog.Error().Msg("Request for unexisting proxy route")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
			return
		default:
			zlog.Error().Err(err).Msg("Failed to query proxy url")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, proxyUrl)
	})

	routes.POST(proxyRoutesEndpoint, func(c *gin.Context) {
		var proxyEndpoint protocol.ProxyEndpoint
		if err := c.Bind(&proxyEndpoint); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().
			Str("path", proxyEndpoint.Path).
			Str("proxy url", proxyEndpoint.ProxyUrl).
			Msg("Received create proxy request")

		if _, err := url.ParseRequestURI(proxyEndpoint.ProxyUrl); err != nil {
			zlog.Error().Err(err).Msg("Failed to parse incoming proxy url")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := database.AddProxyEndpoint(c, proxyEndpoint.Path, proxyEndpoint.ProxyUrl)

		switch err {
		case nil:
			zlog.Info().Str("path", proxyEndpoint.Path).Msg("Proxy endpoint created")
			c.JSON(http.StatusOK, "Proxy endpoint successfully added!")
		case database.ErrDuplicateKey:
			zlog.Error().Str("path", proxyEndpoint.Path).Msg("Endpoint with this path already exists")
			c.JSON(http.StatusConflict, gin.H{"error": "The same endpoint already exists"})
		default:
			zlog.Error().Err(err).Msg("Failed to add proxy endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	routes.PUT(proxyRoutesEndpoint, func(c *gin.Context) {
		var proxyEndpoint protocol.ProxyEndpoint
		if err := c.Bind(&proxyEndpoint); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, err := url.ParseRequestURI(proxyEndpoint.ProxyUrl); err != nil {
			zlog.Error().Err(err).Msg("Failed to parse incoming proxy url")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", proxyEndpoint.Path).Msg("Received update proxy request")

		err := database.UpdateProxyEndpoint(c, proxyEndpoint.Path, proxyEndpoint.ProxyUrl)
		switch err {
		case nil:
			zlog.Info().Str("path", proxyEndpoint.Path).Msg("Proxy endpoint updated")
			c.JSON(http.StatusNoContent, "Proxy endpoint successfully updated!")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Update on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
		default:
			zlog.Error().Err(err).Msg("Failed to add proxy endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	routes.DELETE(proxyRoutesEndpoint, func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received delete proxy request")

		err := database.RemoveProxyEndpoint(c, path)

		switch err {
		case nil:
			zlog.Info().Str("path", path).Msg("Proxy endpoint removed")
			c.JSON(http.StatusNoContent, "Proxy endpoint successfully removed!")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Delete on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
		default:
			zlog.Error().Err(err).Msg("Failed to remove proxy endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})
}

// dynamic routes (each request involves launching user's code)
func (s *server) initRoutesApiDynamic(routes *gin.RouterGroup) {
	dynamicRoutesEndpoint := "/dynamic"

	routes.GET(dynamicRoutesEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all routes dynamic request")

		endpoints, err := database.ListAllDynamicEndpointPaths(c)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to list all dynamic endpoint paths")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
	})

	routes.GET(dynamicRoutesEndpoint+"/code", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			zlog.Error().Msg("Path param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify path param"})
			return
		}

		zlog.Info().Str("path", path).Msg("Received get code dynamic request")

		scriptName, err := database.GetDynamicEndpointScriptName(c, path)
		switch err {
		case nil:
			zlog.Info().Str("script name", scriptName).Msg("Got script")
		case database.ErrNoSuchPath, database.ErrBadRouteType:
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

		c.JSON(http.StatusOK, util.UnwrapCodeForDynHandle(code))
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
				Str("script name", scriptName).
				Msg("Dynamic endpoint added")
			c.JSON(http.StatusOK, "Dynamic endpoint successfully added")
		case database.ErrDuplicateKey:
			zlog.Error().Err(err).Msg("Failed to add dynamic endpoint")
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
		switch err {
		case nil:
			zlog.Info().Str("path", dynamicEndpoint.Path).Msg("Dynamic endpoint updated")
			c.JSON(http.StatusNoContent, "Dynamic endpoint successfully updated")
		case database.ErrNoSuchPath:
			zlog.Error().Msg("Update on unexisting path")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path was not created before"})
			return
		case database.ErrBadRouteType:
			zlog.Error().Msg("Update on route with different type")
			c.JSON(http.StatusNotFound, gin.H{"error": "Received path has different route type"})
			return
		default:
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
			zlog.Error().Err(err).Msg("Failed to dynamic endpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Info().Str("path", path).Msg("Dynamic endpoint removed")
		c.JSON(http.StatusNoContent, "Dynamic endpoint successfully removed")
	})
}

func (s *server) handleStaticRouteRequest(c *gin.Context, route *database.Route) {
	c.JSON(http.StatusOK, route.Response)
}

func (s *server) handleProxyRouteRequest(c *gin.Context, route *database.Route) {
	target, err := url.ParseRequestURI(route.ProxyURL)
	if err != nil {
		zlog.Fatal().
			Err(err).
			Str("proxy url", route.ProxyURL).
			Msg("Failed to parse url on route request!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Method = c.Request.Method

		req.Host = target.Host
		req.URL = target
		if c.Request.URL.RawQuery == "" || target.RawQuery == "" {
			req.URL.RawQuery = c.Request.URL.RawQuery + target.RawQuery
		} else {
			req.URL.RawQuery = c.Request.URL.RawQuery + "&" + target.RawQuery
		}
		req.RequestURI = target.RequestURI()
		zlog.Debug().
			Str("host", req.Host).
			Str("path", req.URL.Path).
			Str("uri", req.RequestURI).
			Str("raw query", req.URL.RawQuery).
			Msg("Request")

		switch req.Method {
		case http.MethodGet, http.MethodHead:
			zlog.Info().Str("method", req.Method).Msg("Skipping body copying because of request method")
		default:
			zlog.Info().Msg("Copying request body")
			var body bytes.Buffer
			{
				defer c.Request.Body.Close()
				if _, err := io.Copy(&body, c.Request.Body); err != nil {
					zlog.Error().Err(err).Msg("Failed to copy request body")
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			req.Body = io.NopCloser(bytes.NewBuffer(body.Bytes()))
		}

		{
			if zlog.Logger.GetLevel() == zerolog.DebugLevel {
				prettyRequest, err := httputil.DumpRequest(req, true)
				if err != nil {
					zlog.Error().Err(err).Msg("Failed to dump request")
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				}
				zlog.Debug().Str("req", string(prettyRequest)).Msg("Rewritten request")
			}
		}
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func (s *server) handleDynamicRouteRequest(c *gin.Context, route *database.Route) {
	worker, err := coderun.WorkerWatcher.BorrowWorker()
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to borrow worker")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer worker.Return()

	headers := c.Request.Header.Clone()
	headersBytes, err := json.Marshal(headers)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to parse headers")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer c.Request.Body.Close()

	output, err := worker.RunScript(FS_CODE_DIR, route.ScriptName, coderun.NewDynHandleArgs(headersBytes, bodyBytes))
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
		zlog.Debug().Interface("route", route).Msg("Queried")

		switch route.Type {
		case database.STATIC_ENDPOINT_TYPE:
			s.handleStaticRouteRequest(c, &route)

		case database.PROXY_ENDPOINT_TYPE:
			s.handleProxyRouteRequest(c, &route)

		case database.DYNAMIC_ENDPOINT_TYPE:
			s.handleDynamicRouteRequest(c, &route)

		default:
			zlog.Fatal().Msg(fmt.Sprintf("Can't resolve route type: %s", route.Type))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Can't resolve route type"})
		}
	})
}

func (s *server) initBrokersApiPool(brokersApi *gin.RouterGroup) {
	poolBrokersEndpoint := "/pool"

	brokersApi.GET(poolBrokersEndpoint, func(c *gin.Context) {
	})

	// list all read and write tasks
	{
		tasks := brokersApi.Group("tasks")

		// list all read tasks by pool name
		tasks.GET("/read", func(c *gin.Context) {
		})

		// list all write tasks by pool name
		tasks.GET("/write", func(c *gin.Context) {})
	}

	brokersApi.POST(poolBrokersEndpoint, func(c *gin.Context) {
		var messagePool protocol.MessagePool
		if err := c.Bind(&messagePool); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch messagePool.Broker {
		case "rabbitmq", "kafka":
			zlog.Info().
				Str("broker", messagePool.Broker).
				Str("queue name", messagePool.QueueName).
				Str("pool name", messagePool.PoolName).
				Msg("Received create pool request")

		default:
			zlog.Error().
				Str("broker", messagePool.Broker).
				Msg("Received request with unsupported broker")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Such pool is unsupported"})
			return
		}

		var pool brokers.MessagePool
		switch messagePool.PoolName {
		case "rabbitmq":
			pool = brokers.NewRabbitMQMessagePool(messagePool.PoolName, messagePool.QueueName)
		case "kafka":
			pool = brokers.NewKafkaMessagePool(messagePool.PoolName, messagePool.QueueName)
		}

		_, err := brokers.AddMessagePool(pool)
		switch err {
		case nil:
			zlog.Info().
				Str("broker", messagePool.Broker).
				Str("pool name", messagePool.PoolName).
				Msg("Pool created")
			c.JSON(http.StatusOK, "Message pool successfully created!")
		case database.ErrDuplicateKey:
			zlog.Error().Err(err).Msg("Failed to add message")
			c.JSON(http.StatusConflict, gin.H{"error": "The same message pool already exists"})
		default:
			zlog.Error().Err(err).Msg("Failed to add message pool")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})

	brokersApi.DELETE(poolBrokersEndpoint, func(c *gin.Context) {
	})
}

func (s *server) initBrokersApiScheduler(brokersApi *gin.RouterGroup) {
	schedulerBrokersEndpoint := "/scheduler"

	// load task messages by task id
	brokersApi.GET(schedulerBrokersEndpoint, func(c *gin.Context) {})

	// schedule read task
	brokersApi.POST(schedulerBrokersEndpoint+"/read", func(c *gin.Context) {
	})

	// schedule write task from protocol.BrokerTask
	brokersApi.POST(schedulerBrokersEndpoint+"/write", func(c *gin.Context) {
	})
}

func (s *server) initBrokersApiEsb(brokersApi *gin.RouterGroup) {
	esbBrokersEndpoint := "/esb"

	// get all tasks by esb pair in-pool name
	brokersApi.GET(esbBrokersEndpoint, func(c *gin.Context) {})

	// create new esb pair
	brokersApi.POST(esbBrokersEndpoint, func(c *gin.Context) {})

	// create delete esb pair
	brokersApi.DELETE(esbBrokersEndpoint, func(c *gin.Context) {})
}
