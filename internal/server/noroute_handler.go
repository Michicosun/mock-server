package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/coderun"
	"mock-server/internal/database"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

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
