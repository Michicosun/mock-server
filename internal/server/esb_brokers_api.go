package server

import (
	"mock-server/internal/brokers"
	"mock-server/internal/database"
	"net/http"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

func (s *server) initBrokersApiEsb(brokersApi *gin.RouterGroup) {
	esbBrokersEndpoint := "/esb"

	// get all tasks by esb pair in-pool name
	brokersApi.GET(esbBrokersEndpoint, func(c *gin.Context) {})

	// create new esb pair
	brokersApi.POST(esbBrokersEndpoint, func(c *gin.Context) {})

	// create delete esb pair
	brokersApi.DELETE(esbBrokersEndpoint, func(c *gin.Context) {
		poolInName := c.Query("pool_in")
		if poolInName == "" {
			zlog.Error().Msg("Pool param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
			return
		}

		err := brokers.RemoveEsbRecord(poolInName)
		switch err {
		case nil:
			zlog.Info().Str("pool", poolInName).Msg("Esb record deleted")
			c.JSON(http.StatusNoContent, "Esb record successfully removed")
		case database.ErrNoSuchRecord:
			zlog.Error().Msg("No such esb record")
			c.JSON(http.StatusNotFound, "No such esb record was created before")
		default:
			zlog.Error().Err(err).Msg("Failed to delete Esb record")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	})
}
