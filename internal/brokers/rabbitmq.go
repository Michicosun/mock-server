package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"mock-server/internal/configs"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	zlog "github.com/rs/zerolog/log"
)

// RabbitMQ base task
type RabbitMQQueueConfig struct {
	Queue      string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       map[string]interface{}
}

type rabbitMQTask struct {
	qcfg *RabbitMQQueueConfig

	conn *amqp.Connection
	ch   *amqp.Channel
	q    *amqp.Queue

	task_id uuid.UUID
}

func (t *rabbitMQTask) set_uuid(id uuid.UUID) {
	t.task_id = id
}

func (t *rabbitMQTask) uuid() uuid.UUID {
	return t.task_id
}

func getConnectionString(s *configs.RabbitMQConnectionConfig) string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", s.Username, s.Password, s.Host, s.Port)
}

func (t *rabbitMQTask) connect_and_prepare() error {
	zlog.Info().Msgf("trying setup connection to rabbitmq")

	s, err := configs.GetRabbitMQConnectionConfig()
	if err != nil {
		return err
	}

	conn, err := amqp.Dial(getConnectionString(s))
	if err != nil {
		return err
	}
	t.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	t.ch = ch

	q, err := ch.QueueDeclare(
		t.qcfg.Queue,
		t.qcfg.Durable,
		t.qcfg.AutoDelete,
		t.qcfg.Exclusive,
		t.qcfg.NoWait,
		t.qcfg.Args,
	)
	if err != nil {
		return err
	}
	t.q = &q

	return nil
}

func (t *rabbitMQTask) close() {
	t.ch.Close()
	t.conn.Close()
}

// RabbitMQ read task
type RabbitMQReadConfig struct {
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      map[string]interface{}
}

type rabbitMQReadTask struct {
	rabbitMQTask
	rcfg *RabbitMQReadConfig

	msgs []amqp.Delivery
}

func (t *rabbitMQReadTask) read(ctx context.Context) error {
	msgs, err := t.ch.Consume(
		t.q.Name,
		t.rcfg.Consumer,
		t.rcfg.AutoAck,
		t.rcfg.Exclusive,
		t.rcfg.NoLocal,
		t.rcfg.NoWait,
		t.rcfg.Args,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgs:
			zlog.Info().Msgf("read msg: %s", string(msg.Body))
			t.msgs = append(t.msgs, msg)
		}
	}
}

func (t *rabbitMQReadTask) json() ([]byte, error) {
	return json.Marshal(t.msgs)
}

// RabbitMQ write task
type RabbitMQWriteConfig struct {
	Exchange    string
	Mandatory   bool
	Immediate   bool
	ContentType string
}

type rabbitMQWriteTask struct {
	rabbitMQTask
	wcfg *RabbitMQWriteConfig

	msgs [][]byte
}

func (t *rabbitMQWriteTask) write(ctx context.Context) error {
	zlog.Info().Msgf("preparing to write %d msgs", len(t.msgs))

	for _, msg := range t.msgs {
		err := t.ch.PublishWithContext(ctx,
			t.wcfg.Exchange,
			t.q.Name,
			t.wcfg.Mandatory,
			t.wcfg.Immediate,
			amqp.Publishing{
				ContentType: t.wcfg.ContentType,
				Body:        msg,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// sugar

func newRabbitMQConnection(queue string) rabbitMQTask {
	return rabbitMQTask{
		qcfg: &RabbitMQQueueConfig{
			Queue:      queue,
			Durable:    false,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		},
	}
}

func NewRabbitMQReadTask(queue string) *rabbitMQReadTask {
	return &rabbitMQReadTask{
		rabbitMQTask: newRabbitMQConnection(queue),
		rcfg: &RabbitMQReadConfig{
			Consumer:  "",
			AutoAck:   true,
			Exclusive: false,
			NoLocal:   false,
			NoWait:    false,
			Args:      nil,
		},
	}
}

func NewRabbitMQWriteTask(queue string) *rabbitMQWriteTask {
	return &rabbitMQWriteTask{
		rabbitMQTask: newRabbitMQConnection(queue),
		wcfg: &RabbitMQWriteConfig{
			Exchange:    "",
			Mandatory:   false,
			Immediate:   false,
			ContentType: "text/plain",
		},
		msgs: [][]byte{},
	}
}

func (t *rabbitMQTask) SetQueueConfig(cfg RabbitMQQueueConfig) *rabbitMQTask {
	t.qcfg = &cfg
	return t
}

func (t *rabbitMQReadTask) SetReadConfig(cfg RabbitMQReadConfig) *rabbitMQReadTask {
	t.rcfg = &cfg
	return t
}

func (t *rabbitMQWriteTask) SetWriteConfig(cfg RabbitMQWriteConfig) *rabbitMQWriteTask {
	t.wcfg = &cfg
	return t
}

func (t *rabbitMQReadTask) Read() {
	BrokerPool.SubmitReadTask(t)
}

func (t *rabbitMQWriteTask) Write(msgs [][]byte) {
	t.msgs = msgs
	BrokerPool.SubmitWriteTask(t)
}
