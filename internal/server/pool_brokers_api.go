package server

import (
	"mock-server/internal/brokers"
	"mock-server/internal/database"
	"mock-server/internal/server/protocol"
	"net/http"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

func (s *server) initBrokersApiPool(brokersApi *gin.RouterGroup) {
	poolBrokersEndpoint := "/pool"

	brokersApi.GET(poolBrokersEndpoint, func(c *gin.Context) {
	})

	// list all read and write tasks
	{
		pool := brokersApi.Group(poolBrokersEndpoint)

		// list all read messages by pool name
		pool.GET("/read", func(c *gin.Context) {})

		// list all write messages by pool name
		pool.GET("/write", func(c *gin.Context) {})

		// start read task in given pool
		pool.POST("/read", func(c *gin.Context) {})

		// start write task in given pool with given messages
		pool.POST("/write", func(c *gin.Context) {})
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
