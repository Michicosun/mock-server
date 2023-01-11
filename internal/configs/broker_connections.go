package configs

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

type UndefinedConnection struct {
	broker string
}

func (e *UndefinedConnection) Error() string {
	return fmt.Sprintf("connection config to broker: %s is not provided", e.broker)
}

type RabbitMQConnectionConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type BrokerConnections struct {
	Rabbitmq RabbitMQConnectionConfig `yaml:"rabbitmq"`
}

var rabbitmq_init sync.Once

func RabbitMQConnectionConfigInit() {
	host := os.Getenv("RABBITMQ_HOST")
	if host != "" {
		config.Brokers.Rabbitmq.Host = host
	}

	s := os.Getenv("RABBITMQ_PORT")
	port, err := strconv.Atoi(s)
	if err == nil {
		config.Brokers.Rabbitmq.Port = port
	}

	username := os.Getenv("RABBITMQ_USERNAME")
	if username != "" {
		config.Brokers.Rabbitmq.Username = username
	}

	password := os.Getenv("RABBITMQ_PASSWORD")
	if password != "" {
		config.Brokers.Rabbitmq.Password = password
	}
}

func GetRabbitMQConnectionConfig() (*RabbitMQConnectionConfig, error) {
	rabbitmq_init.Do(RabbitMQConnectionConfigInit)

	if config.Brokers.Rabbitmq.Host == "" {
		return nil, &UndefinedConnection{
			broker: "RabbitMQ",
		}
	}

	return &config.Brokers.Rabbitmq, nil
}
