package main

import (
	"context"
	"fmt"
	"mock-server-backend/internal/brokers"
	"time"

	filename "github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
)

func init_logger() {
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "Jan _2 15:04:05.000000000"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true

	hook := filename.NewHook()
	hook.Field = "source"
	log.AddHook(hook)
	log.SetFormatter(Formatter)
	log.SetLevel(log.InfoLevel)
}

func main() {
	init_logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info("starting...")

	brokers.BrokerPool.Init(ctx, brokers.BPoolConfig{
		R_workers:     50,
		W_workers:     50,
		Read_timeout:  10 * time.Second,
		Write_timeout: 10 * time.Second,
		Disable_task:  5 * time.Second,
	})

	brokers.BrokerPool.Start()

	secret_id, _ := brokers.SecretBox.SetSecret(&brokers.RabbitMQSecret{
		Username: "guest",
		Password: "guest",
		Host:     "localhost",
		Port:     5672,
		Queue:    "test-mock-queue",
	})

	conn := brokers.NewRabbitMQConnection(secret_id)

	conn.Write([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Submit()

	<-time.After(1 * time.Second)

	conn.Read().Submit()

	log.Info("start reading")
	<-time.After(20 * time.Second)

	cancel()

	brokers.BrokerPool.Wait()
}
