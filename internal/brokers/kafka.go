package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/database"
	"sync/atomic"
	"time"

	kafka "github.com/confluentinc/confluent-kafka-go/kafka"
	zlog "github.com/rs/zerolog/log"
)

type KafkaTopicConfig struct {
	Addr     string `json:"addr"`
	ClientId string `json:"client_id"`
	GroupId  string `json:"group_id"`
}

type KafkaReadConfig struct {
	OffsetReset string `json:"offset_reset"`
}

type KafkaWriteConfig struct {
	Acks string `json:"acks"`
}

type KafkaMessagePoolConfig struct {
	Topic string           `json:"topic"`
	Tcfg  KafkaTopicConfig `json:"tcfg"`
	Rcfg  KafkaReadConfig  `json:"rcfg"`
	Wcfg  KafkaWriteConfig `json:"wcfg"`
}

type KafkaMessagePool struct {
	name  string
	topic string
	tcfg  *KafkaTopicConfig
	rcfg  *KafkaReadConfig
	wcfg  *KafkaWriteConfig
}

func (mp *KafkaMessagePool) GetName() string {
	return mp.name
}

func (mp *KafkaMessagePool) GetQueue() string {
	return mp.topic
}

func (mp *KafkaMessagePool) GetBroker() string {
	return "kafka"
}

func (mp *KafkaMessagePool) GetConfig() interface{} {
	return &KafkaMessagePoolConfig{
		Topic: mp.topic,
		Tcfg:  *mp.tcfg,
		Rcfg:  *mp.rcfg,
		Wcfg:  *mp.wcfg,
	}
}

func (mp *KafkaMessagePool) GetJSONConfig() ([]byte, error) {
	return json.Marshal(mp.GetConfig())
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

func (t *kafkaTask) connectAndPrepare(ctx context.Context) error {
	zlog.Info().Str("addr", t.pool.tcfg.Addr).Msg("using addr for kafka connection")
	return nil
}

func (t *kafkaTask) close() {
	zlog.Info().Msg("kafka connection closed")
}

type kafkaReadTask struct {
	kafkaTask
	msgs []string
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
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{t.pool.topic}, nil); err != nil {
		return err
	}

	has_esb_record := true
	esb_record, err := database.GetESBRecord(ctx, t.pool.name)
	if err == database.ErrNoSuchRecord {
		has_esb_record = false
	} else if err != nil {
		return err
	}

	err = nil
	run := atomic.Bool{}
	run.Store(true)
	read_canceled := make(chan struct{}, 1)

	go func() {
		for run.Load() {
			ev := consumer.Poll(500)
			switch e := ev.(type) {
			case *kafka.Message:
				zlog.Info().Str("task", string(t.getTaskId())).Str("msg", string(e.Value)).Msg("get message")
				t.msgs = append(t.msgs, string(e.Value))
				if databaseErr := database.AddTaskMessage(context.TODO(), database.TaskMessage{
					TaskId:  string(t.getTaskId()),
					Message: string(e.Value),
				}); err != nil {
					zlog.Err(err).Msg(fmt.Sprintf("Failed to upload message for task: %s", t.getTaskId()))
					err = databaseErr
					run.Store(false)
				}
			case kafka.Error:
				err = e
				run.Store(false)
			default:
				if has_esb_record && len(t.msgs) > 0 {
					if esb_err := submitToESB(esb_record, t.msgs); esb_err != nil {
						err = esb_err
						run.Store(false)
					}

					t.msgs = make([]string, 0)
				}
			}
		}
		zlog.Info().Str("task", string(t.getTaskId())).Msg("read canceled")
		read_canceled <- struct{}{}
	}()

	zlog.Info().Str("task", string(t.getTaskId())).Msg("waiting for read deadline")
	<-ctx.Done()

	run.Store(false)
	<-read_canceled

	return err
}

// Kafka write task
type kafkaWriteTask struct {
	kafkaTask
	msgs []string
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
	defer producer.Close()

	delivery_chan := make(chan kafka.Event, len(t.msgs))

	for _, msg := range t.msgs {
		err = producer.Produce(
			&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &t.pool.topic,
					Partition: kafka.PartitionAny,
				},
				Value: []byte(msg),
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
				if err = database.AddTaskMessage(context.TODO(), database.TaskMessage{
					TaskId:  string(t.getTaskId()),
					Message: string(m.Value),
				}); err != nil {
					zlog.Err(err).Msg(fmt.Sprintf("Failed to upload message for task: %s", t.getTaskId()))
					return err
				}
			}
		}
	}

	close(delivery_chan)

	return nil
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

func createKafkaPoolFromDatabase(pool database.MessagePool) (*KafkaMessagePool, error) {
	var config KafkaMessagePoolConfig
	err := json.Unmarshal([]byte(pool.Config), &config)
	if err != nil {
		return nil, err
	}
	newPool := NewKafkaMessagePool(pool.Name, config.Topic)
	newPool.tcfg = &config.Tcfg
	newPool.wcfg = &config.Wcfg
	newPool.rcfg = &config.Rcfg
	return newPool, nil
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

func (mp *KafkaMessagePool) NewReadTask() qReadTask {
	return &kafkaReadTask{
		kafkaTask: newKafkaBaseTask(mp),
	}
}

func (mp *KafkaMessagePool) NewWriteTask(data []string) qWriteTask {
	return &kafkaWriteTask{
		kafkaTask: newKafkaBaseTask(mp),
		msgs:      data,
	}
}

func getKafkaConnectionString(s *configs.KafkaConnectionConfig) string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (mp *KafkaMessagePool) CreateBrokerEndpoint() error {
	zlog.Info().Str("pool", mp.name).Str("broker", mp.GetBroker()).Msg("preparing to create pool")

	cfg, err := configs.GetKafkaConnectionConfig()
	if err != nil {
		return err
	}

	mp.tcfg.Addr = getKafkaConnectionString(cfg)
	mp.tcfg.ClientId = cfg.ClientId
	mp.tcfg.GroupId = cfg.GroupId

	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": mp.tcfg.Addr})
	if err != nil {
		return err
	}

	defer admin.Close()

	results, err := admin.CreateTopics(
		context.TODO(),
		[]kafka.TopicSpecification{{
			Topic:         mp.topic,
			NumPartitions: 1,
		}},
		kafka.SetAdminOperationTimeout(time.Second*60),
	)
	if err != nil {
		return err
	}

	for _, result := range results {
		zlog.Info().Str("topic", result.Topic).Msg("create topic")
	}

	zlog.Info().Str("pool", mp.name).Str("broker", mp.GetBroker()).Msg("pool was created")

	return err

}

func (mp *KafkaMessagePool) RemoveBrokerEndpoint() error {
	zlog.Info().Str("pool", mp.name).Str("broker", mp.GetBroker()).Msg("preparing to remove pool")

	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": mp.tcfg.Addr})
	if err != nil {
		return err
	}

	defer admin.Close()

	results, err := admin.DeleteTopics(context.TODO(), []string{mp.topic}, kafka.SetAdminOperationTimeout(time.Second*60))
	if err != nil {
		return err
	}

	for _, result := range results {
		zlog.Info().Str("topic", result.Topic).Msg("delete topic")
	}

	zlog.Info().Str("pool", mp.name).Str("broker", mp.GetBroker()).Msg("pool was deleted")

	return nil
}
