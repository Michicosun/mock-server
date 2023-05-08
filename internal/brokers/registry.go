package brokers

import (
	"context"
	"fmt"
	"mock-server/internal/database"
)

type MessagePool interface {
	getName() string
	getBroker() string
	getJSONConfig() ([]byte, error)
	NewReadTask() qReadTask
	NewWriteTask(data []string) qWriteTask
}

func createFromDatabase(pool database.MessagePool) (MessagePool, error) {
	if pool.Broker == "rabbitmq" {
		return createRabbitMQPoolFromDatabase(pool)
	} else {
		return createKafkaPoolFromDatabase(pool)
	}
}

func AddMessagePool(pool MessagePool) (MessagePool, error) {
	jsonConfig, err := pool.getJSONConfig()
	if err != nil {
		return nil, err
	}
	err = database.AddMessagePool(context.TODO(), database.MessagePool{
		Name:   pool.getName(),
		Broker: pool.getBroker(),
		Config: jsonConfig,
	})
	if err == database.ErrDuplicateKey {
		return nil, fmt.Errorf("pool: %s is already registered", pool.getName())
	}

	return pool, err
}

func RemoveMessagePool(poolName string) error {
	err := database.RemoveMessagePool(context.TODO(), poolName)
	if err == database.ErrNoSuchPath {
		return fmt.Errorf("pool: %s is not registered", poolName)
	}

	return err
}

func GetMessagePool(poolName string) (MessagePool, error) {
	pool, err := database.GetMessagePool(context.TODO(), poolName)
	if err == database.ErrNoSuchPath {
		return nil, fmt.Errorf("pool: %s is not registered", poolName)
	} else if err != nil {
		return nil, err
	}

	return createFromDatabase(pool)
}
