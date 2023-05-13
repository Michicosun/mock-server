package server

import (
	"mock-server/internal/database"
	"mock-server/internal/server/protocol"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

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
