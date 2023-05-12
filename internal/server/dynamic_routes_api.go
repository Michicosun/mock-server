package server

import (
	"mock-server/internal/database"
	"mock-server/internal/server/protocol"
	"mock-server/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

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
