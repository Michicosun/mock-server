package brokers

import (
	"context"
	"errors"
	"mock-server/internal/database"
)

type MessagePool interface {
	GetName() string
	GetQueue() string
	GetBroker() string
	GetJSONConfig() ([]byte, error)
	NewReadTask() qReadTask
	NewWriteTask(data []string) qWriteTask
}

func createFromDatabase(pool database.MessagePool) (MessagePool, error) {
	switch pool.Broker {
	case "rabbitmq":
		return createRabbitMQPoolFromDatabase(pool)
	case "kafka":
		return createKafkaPoolFromDatabase(pool)
	default:
		return nil, errors.New("invalid message pool type")
	}
}

func AddMessagePool(pool MessagePool) (MessagePool, error) {
	jsonConfig, err := pool.GetJSONConfig()
	if err != nil {
		return nil, err
	}
	err = database.AddMessagePool(context.TODO(), database.MessagePool{
		Name:   pool.GetName(),
		Broker: pool.GetBroker(),
		Config: jsonConfig,
	})

	return pool, err
}

func RemoveMessagePool(poolName string) error {
	err := database.RemoveMessagePool(context.TODO(), poolName)
	return err
}

func GetMessagePool(poolName string) (MessagePool, error) {
	pool, err := database.GetMessagePool(context.TODO(), poolName)
	if err != nil {
		return nil, err
	}
	return createFromDatabase(pool)
}
