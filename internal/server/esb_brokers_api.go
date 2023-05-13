package server

import (
	"mock-server/internal/brokers"
	"mock-server/internal/database"
	"mock-server/internal/server/protocol"
	"mock-server/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

func (s *server) initBrokersApiEsb(brokersApi *gin.RouterGroup) {
	esbBrokersEndpoint := "/esb"

	// get all esb records
	brokersApi.GET(esbBrokersEndpoint, func(c *gin.Context) {
		zlog.Info().Msg("Get all esb records request")

		esbRecords, err := database.ListESBRecords(c)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to list all esb records")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		respEsbRecords := make([]protocol.EsbRecord, 0)
		for _, esbRecord := range esbRecords {
			respEsbRecords = append(respEsbRecords, protocol.EsbRecord{
				PoolNameIn:  esbRecord.PoolNameIn,
				PoolNameOut: esbRecord.PoolNameOut,
			})
		}

		zlog.Debug().Interface("records", respEsbRecords).Msg("Successfully queried all esb records")
		c.JSON(http.StatusOK, gin.H{"records": respEsbRecords})
	})

	// get mapper code for esb record by in-pool name
	brokersApi.GET(esbBrokersEndpoint+"/code", func(c *gin.Context) {
		poolInName := c.Query("pool_in")
		if poolInName == "" {
			zlog.Error().Msg("Pool param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
			return
		}

		esbRecord, err := brokers.GetEsbRecord(c, poolInName)
		switch err {
		case nil:
			zlog.Info().Str("pool", esbRecord.PoolNameIn).Msg("Queried")
			if esbRecord.MapperScriptName == brokers.EMPTY_MAPPER {
				zlog.Error().Str("pool", esbRecord.PoolNameIn).Msg("No code for esb record")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Such record does not have mapper code"})
				return
			}
		case database.ErrNoSuchRecord:
			zlog.Error().Str("pool", esbRecord.PoolNameIn).Msg("Such record was not created before")
			c.JSON(http.StatusNotFound, gin.H{"error": "Such record was not created before"})
			return
		default:
			zlog.Error().Err(err).Msg("Failed to get esb record")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		code, err := s.fs.Read(FS_ESB_DIR, esbRecord.MapperScriptName)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to read esb record code")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, util.UnwrapCodeForEsb(code))
	})

	// create new esb pair
	brokersApi.POST(esbBrokersEndpoint, func(c *gin.Context) {
		var esbRecord protocol.EsbRecord
		if err := c.Bind(&esbRecord); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch esbRecord.Code {
		case "":
			zlog.Info().
				Str("pool in", esbRecord.PoolNameIn).
				Str("pool out", esbRecord.PoolNameOut).
				Msg("Received create request for esb record without code")

			err := brokers.AddEsbRecord(c, esbRecord.PoolNameIn, esbRecord.PoolNameOut)
			switch err {
			case nil:
				zlog.Info().
					Str("pool in", esbRecord.PoolNameIn).
					Str("pool out", esbRecord.PoolNameOut).
					Msg("Esb record added")
				c.JSON(http.StatusOK, "Esb record successfully added!")
			case database.ErrDuplicateKey:
				zlog.Error().Msg("Esb record with the same in-pool already exists")
				c.JSON(http.StatusConflict, gin.H{"error": "Esb record with the same in-pool already exists"})
			default:
				zlog.Error().Err(err).Msg("Failed to add esb record")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
		default:
			zlog.Info().
				Str("pool in", esbRecord.PoolNameIn).
				Str("pool out", esbRecord.PoolNameOut).
				Msg("Received create request for esb record with code")

			scriptName := util.GenUniqueFilename("py")
			if err := s.fs.Write(FS_ESB_DIR, scriptName, util.WrapCodeForEsb(esbRecord.Code)); err != nil {
				zlog.Error().Err(err).Msg("Failed to create script file")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			err := brokers.AddEsbRecordWithMapper(c, esbRecord.PoolNameIn, esbRecord.PoolNameOut, scriptName)
			switch err {
			case nil:
				zlog.Info().
					Str("pool in", esbRecord.PoolNameIn).
					Str("pool out", esbRecord.PoolNameOut).
					Str("mapper script name", scriptName).
					Msg("Esb record added")
				c.JSON(http.StatusOK, "Esb record successfully added!")
			case database.ErrDuplicateKey:
				zlog.Error().Msg("Esb record with the same in-pool already exists")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Esb record with the same in-pool already exists"})
			default:
				zlog.Error().Err(err).Msg("Failed to add esb record")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
		}
	})

	// create delete esb pair
	brokersApi.DELETE(esbBrokersEndpoint, func(c *gin.Context) {
		poolInName := c.Query("pool_in")
		if poolInName == "" {
			zlog.Error().Msg("Pool param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
			return
		}

		err := brokers.RemoveEsbRecord(c, poolInName)
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
