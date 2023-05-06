package brokers

import (
	"context"
	"fmt"
	"mock-server/internal/configs"
	"sync/atomic"

	kafka "github.com/confluentinc/confluent-kafka-go/kafka"
	zlog "github.com/rs/zerolog/log"
)

type KafkaTopicConfig struct {
	Addr     string
	ClientId string
	GroupId  string
}

type KafkaReadConfig struct {
	OffsetReset string
}

type KafkaWriteConfig struct {
	Acks string
}

type KafkaMessagePool struct {
	name  string
	topic string
	tcfg  *KafkaTopicConfig
	rcfg  *KafkaReadConfig
	wcfg  *KafkaWriteConfig
}

type kafkaMessagePoolHandler struct {
	pool *KafkaMessagePool
}

func (mp *KafkaMessagePool) getName() string {
	return mp.name
}

func (mp *KafkaMessagePool) getBroker() string {
	return "kafka"
}

func (mp *KafkaMessagePool) getHandler() MessagePoolHandler {
	return &kafkaMessagePoolHandler{
		pool: mp,
	}
}

// Kafka base task
type kafkaTask struct {
	pool *KafkaMessagePool
}

func (t *kafkaTask) getMessagePool() MessagePool {
	return t.pool
}

func (t *kafkaTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("kafka:%s:%s", t.pool.name, t.pool.topic))
}

func (t *kafkaTask) getConnectionString(s *configs.KafkaConnectionConfig) string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (t *kafkaTask) connectAndPrepare() error {
	zlog.Info().Str("task", string(t.getTaskId())).Msg("configuring connection to kafka")

	cfg, err := configs.GetKafkaConnectionConfig()
	if err != nil {
		return err
	}

	t.pool.tcfg.Addr = t.getConnectionString(cfg)
	t.pool.tcfg.ClientId = cfg.ClientId
	t.pool.tcfg.GroupId = cfg.GroupId

	zlog.Info().Str("addr", t.pool.tcfg.Addr).Msg("using addr for kafka connection")

	return err
}

func (t *kafkaTask) close() {
	zlog.Info().Msg("kafka connection closed")
}

type kafkaReadTask struct {
	kafkaTask
	msgs []*kafka.Message
}

func (t *kafkaReadTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("%s:read", t.kafkaTask.getTaskId()))
}

func (t *kafkaReadTask) Schedule() TaskId {
	return MPTaskScheduler.submitReadTask(t)
}

func (t *kafkaReadTask) read(ctx context.Context) error {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": t.pool.tcfg.Addr,
		"group.id":          t.pool.tcfg.GroupId,
		"auto.offset.reset": t.pool.rcfg.OffsetReset,
	})

	if err != nil {
		return err
	}

	if err := consumer.SubscribeTopics([]string{t.pool.topic}, nil); err != nil {
		return err
	}

	err = nil
	run := atomic.Bool{}
	run.Store(true)
	read_canceled := make(chan struct{}, 1)

	go func() {
		for run.Load() {
			ev := consumer.Poll(100)
			switch e := ev.(type) {
			case *kafka.Message:
				zlog.Info().Str("task", string(t.getTaskId())).Str("msg", string(e.Value)).Msg("get message")
				t.msgs = append(t.msgs, e)
			case kafka.Error:
				err = e
				run.Store(false)
			}
		}
		zlog.Info().Str("task", string(t.getTaskId())).Msg("read canceled")
		read_canceled <- struct{}{}
	}()

	zlog.Info().Str("task", string(t.getTaskId())).Msg("waiting for read deadline")
	<-ctx.Done()

	run.Store(false)
	consumer.Close()
	<-read_canceled

	return err
}

func (t *kafkaReadTask) messages() ([][]byte, error) {
	msgs := make([][]byte, 0)

	for _, msg := range t.msgs {
		msgs = append(msgs, msg.Value)
	}

	return msgs, nil
}

// Kafka write task
type kafkaWriteTask struct {
	kafkaTask
	msgs [][]byte
}

func (t *kafkaWriteTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("%s:write", t.kafkaTask.getTaskId()))
}

func (t *kafkaWriteTask) Schedule() TaskId {
	return MPTaskScheduler.submitWriteTask(t)
}

func (t *kafkaWriteTask) write(ctx context.Context) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": t.pool.tcfg.Addr,
		"client.id":         t.pool.tcfg.ClientId,
		"acks":              t.pool.wcfg.Acks,
	})

	if err != nil {
		return err
	}

	delivery_chan := make(chan kafka.Event, len(t.msgs))

	for _, msg := range t.msgs {
		err = producer.Produce(
			&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &t.pool.topic,
					Partition: kafka.PartitionAny,
				},
				Value: msg,
			},
			delivery_chan,
		)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(t.msgs); i += 1 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout to write batch")
		case e := <-delivery_chan:
			m := e.(*kafka.Message)

			if m.TopicPartition.Error != nil {
				return m.TopicPartition.Error
			} else {
				zlog.Info().Str("task", string(t.getTaskId())).Msgf(
					"Delivered message to topic %s [%d] at offset %v",
					*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset,
				)
			}
		}
	}

	close(delivery_chan)

	return nil
}

func (t *kafkaWriteTask) messages() [][]byte {
	return t.msgs
}

func NewKafkaMessagePool(name string, topic string) *KafkaMessagePool {
	return &KafkaMessagePool{
		name:  name,
		topic: topic,
		tcfg:  &KafkaTopicConfig{},
		rcfg: &KafkaReadConfig{
			OffsetReset: "smallest",
		},
		wcfg: &KafkaWriteConfig{
			Acks: "all",
		},
	}
}

func (mp *KafkaMessagePool) SetReadConfig(cfg KafkaReadConfig) *KafkaMessagePool {
	mp.rcfg = &cfg
	return mp
}

func (mp *KafkaMessagePool) SetWriteConfig(cfg KafkaWriteConfig) *KafkaMessagePool {
	mp.wcfg = &cfg
	return mp
}

func newKafkaBaseTask(pool *KafkaMessagePool) kafkaTask {
	return kafkaTask{
		pool: pool,
	}
}

func (h *kafkaMessagePoolHandler) NewReadTask() qReadTask {
	return &kafkaReadTask{
		kafkaTask: newKafkaBaseTask(h.pool),
	}
}

func (h *kafkaMessagePoolHandler) NewWriteTask(data [][]byte) qWriteTask {
	return &kafkaWriteTask{
		kafkaTask: newKafkaBaseTask(h.pool),
		msgs:      data,
	}
}
