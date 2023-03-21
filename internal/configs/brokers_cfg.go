package configs

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
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

type MPTaskSchedulerConfig struct {
	R_workers     uint32        `yaml:"r_workers"`
	W_workers     uint32        `yaml:"w_workers"`
	Read_timeout  time.Duration `yaml:"read_timeout"`
	Write_timeout time.Duration `yaml:"write_timeout"`
}

type BrokersConfig struct {
	Scheduler MPTaskSchedulerConfig    `yaml:"scheduler"`
	Rabbitmq  RabbitMQConnectionConfig `yaml:"rabbitmq"`
}

func GetMPTaskSchedulerConfig() *MPTaskSchedulerConfig {
	return &config.Brokers.Scheduler
}

var rabbitmq_init sync.Once

func RabbitMQConnectionConfigInit() {
	host, ok := os.LookupEnv("RABBITMQ_HOST")
	if ok {
		config.Brokers.Rabbitmq.Host = host
	}

	s, ok := os.LookupEnv("RABBITMQ_PORT")
	if port, err := strconv.Atoi(s); ok && err != nil {
		config.Brokers.Rabbitmq.Port = port
	}

	username, ok := os.LookupEnv("RABBITMQ_USERNAME")
	if ok {
		config.Brokers.Rabbitmq.Username = username
	}

	password, ok := os.LookupEnv("RABBITMQ_PASSWORD")
	if ok {
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
