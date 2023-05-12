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
		zlog.Info().Msg("Get all broker pools request")

		pools, err := database.ListMessagePools(c)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to list all broker pools")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		respPools := make([]protocol.MessagePool, 0)
		for _, pool := range pools {
			switch pool.Broker {
			case "rabbitmq":
				respPools = append(respPools, protocol.MessagePool{
					PoolName:  pool.Name,
					QueueName: pool.Queue,
					Broker:    "rabbitmq",
				})
			case "kafka":
				respPools = append(respPools, protocol.MessagePool{
					PoolName:  pool.Name,
					TopicName: pool.Queue,
					Broker:    "kafka",
				})

			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database inconsistency found"})
			}
		}

		zlog.Debug().Interface("pools", respPools).Msg("Successfully queried all pools")
		c.JSON(http.StatusOK, gin.H{"pools": respPools})
	})

	brokersApi.GET(poolBrokersEndpoint+"/config", func(c *gin.Context) {
		poolName := c.Query("pool")
		if poolName == "" {
			zlog.Error().Msg("Pool param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
			return
		}

		pool, err := brokers.GetMessagePool(poolName)
		switch err {
		case nil:
			zlog.Info().Str("pool", poolName).Msg("Queried pool")
		case database.ErrNoSuchPath:
			zlog.Error().Str("pool", poolName).Msg("No such pool")
			c.JSON(http.StatusNotFound, gin.H{"error": "No such pool"})
			return
		default:
			zlog.Error().Err(err).Msg("Failed to get pool")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		config, err := pool.GetJSONConfig()
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to get pool config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		zlog.Debug().Interface("config", config).Msg("Parsed config")
		c.JSON(http.StatusOK, config)
	})

	// list all read and write tasks
	{
		pool := brokersApi.Group(poolBrokersEndpoint)

		// list all read messages by pool name
		pool.GET("/read", func(c *gin.Context) {
			poolName := c.Query("pool")
			if poolName == "" {
				zlog.Error().Msg("Pool param not specified")
				c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
				return
			}

			zlog.Info().Str("pool", poolName).Msg("Received pool read tasks list request")

			pool, err := database.GetMessagePool(c, poolName)
			switch err {
			case nil:
				zlog.Info().Str("pool", poolName).Msg("Queried pool")
			case database.ErrNoSuchPath:
				zlog.Error().Str("pool", poolName).Msg("No such pool")
				c.JSON(http.StatusNotFound, gin.H{"error": "No such pool"})
				return
			default:
				zlog.Error().Err(err).Msg("Failed to get pool")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			messages, err := database.GetMessagePoolReadMessages(c, pool)
			if err != nil {
				zlog.Error().Err(err).Msg("Failed to get pool read messages")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			zlog.Debug().Interface("messages", messages).Msg("Queried messages")
			c.JSON(http.StatusOK, gin.H{"messages": messages})
		})

		// list all write messages by pool name
		pool.GET("/write", func(c *gin.Context) {
			poolName := c.Query("pool")
			if poolName == "" {
				zlog.Error().Msg("Pool param not specified")
				c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
				return
			}

			zlog.Info().Str("pool", poolName).Msg("Received pool write tasks list request")

			pool, err := database.GetMessagePool(c, poolName)
			switch err {
			case nil:
				zlog.Info().Str("pool", poolName).Msg("Queried pool")
			case database.ErrNoSuchPath:
				zlog.Error().Str("pool", poolName).Msg("No such pool")
				c.JSON(http.StatusNotFound, gin.H{"error": "No such pool"})
				return
			default:
				zlog.Error().Err(err).Msg("Failed to get pool")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			messages, err := database.GetMessagePoolWriteMessages(c, pool)
			if err != nil {
				zlog.Error().Err(err).Msg("Failed to get pool write messages")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			zlog.Debug().Interface("messages", messages).Msg("Queried messages")
			c.JSON(http.StatusOK, gin.H{"messages": messages})
		})

		// start read task in given pool
		pool.POST("/read", func(c *gin.Context) {
			poolName := c.Query("pool")
			if poolName == "" {
				zlog.Error().Msg("Pool param not specified")
				c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
				return
			}

			zlog.Info().Str("pool", poolName).Msg("Received pool read task")

			pool, err := brokers.GetMessagePool(poolName)
			switch err {
			case nil:
				zlog.Info().Str("pool", pool.GetName()).Msg("Queried pool")
			case database.ErrNoSuchPath:
				zlog.Error().Str("pool", pool.GetName()).Msg("No such pool")
				c.JSON(http.StatusNotFound, gin.H{"error": "No such pool"})
				return
			default:
				zlog.Error().Err(err).Msg("Failed to get pool")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			zlog.Debug().Str("pool", pool.GetName()).Msg("Scheduling new read task")
			pool.NewReadTask().Schedule()
		})

		// start write task in given pool with given messages
		pool.POST("/write", func(c *gin.Context) {
			var brokerTask protocol.BrokerTask
			if err := c.Bind(&brokerTask); err != nil {
				zlog.Error().Err(err).Msg("Failed to bind request")
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			zlog.Info().Str("pool", brokerTask.PoolName).Msg("Received pool write task")

			pool, err := brokers.GetMessagePool(brokerTask.PoolName)
			switch err {
			case nil:
				zlog.Info().Str("pool", pool.GetName()).Msg("Queried pool")
			case database.ErrNoSuchPath:
				zlog.Error().Str("pool", pool.GetName()).Msg("No such pool")
				c.JSON(http.StatusNotFound, gin.H{"error": "No such pool"})
				return
			default:
				zlog.Error().Err(err).Msg("Failed to get pool")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			zlog.Debug().Str("pool", pool.GetName()).Msg("Schedulting new write task")
			pool.NewWriteTask(brokerTask.Messages).Schedule()
		})
	}

	brokersApi.POST(poolBrokersEndpoint, func(c *gin.Context) {
		var messagePool protocol.MessagePool
		if err := c.Bind(&messagePool); err != nil {
			zlog.Error().Err(err).Msg("Failed to bind request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch messagePool.Broker {
		case "rabbitmq":
			if messagePool.QueueName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "queue_name field required for rabbitmq pool"})
				return
			}
			zlog.Info().
				Str("broker", messagePool.Broker).
				Str("queue name", messagePool.QueueName).
				Str("pool name", messagePool.PoolName).
				Msg("Received create pool request")
		case "kafka":
			if messagePool.TopicName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "topic_name field required for kafka pool"})
				return
			}
			zlog.Info().
				Str("broker", messagePool.Broker).
				Str("topic name", messagePool.TopicName).
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
		switch messagePool.Broker {
		case "rabbitmq":
			pool = brokers.NewRabbitMQMessagePool(messagePool.PoolName, messagePool.QueueName)
		case "kafka":
			pool = brokers.NewKafkaMessagePool(messagePool.PoolName, messagePool.TopicName)
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
		poolName := c.Query("pool")
		if poolName == "" {
			zlog.Error().Msg("Pool param not specified")
			c.JSON(http.StatusBadRequest, gin.H{"error": "specify pool param"})
			return
		}

		if err := brokers.RemoveMessagePool(poolName); err != nil {
			zlog.Error().Err(err).Msg("Failed to remove pool")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusNoContent, "Message pool successfully removed")
	})
}
