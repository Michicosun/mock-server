package server

import (
	"mock-server/internal/database"
	"mock-server/internal/server/protocol"
	"net/http"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

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
