package brokers

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

// RabbitMQ secret
type RabbitMQSecret struct {
	Username string
	Password string
	Host     string
	Port     int
	Queue    string
}

func (s *RabbitMQSecret) GetSecretId() SecretId {
	return SecretId(fmt.Sprintf("amqp://%s:%d/?queue=%s", s.Host, s.Port, s.Queue))
}

func (s *RabbitMQSecret) GetConnectionString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", s.Username, s.Password, s.Host, s.Port)
}

// RabbitMQ base task
type RabbitMQQueueConfig struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       map[string]interface{}
}

type rabbitMQTask struct {
	secret_id SecretId
	qcfg      *RabbitMQQueueConfig

	conn *amqp.Connection
	ch   *amqp.Channel
	q    *amqp.Queue
}

func (t *rabbitMQTask) connect_and_prepare() error {
	s, err := SecretBox.GetSecret(t.secret_id)
	if err != nil {
		return err
	}

	rabbit_s, ok := s.(*RabbitMQSecret)
	if !ok {
		return &WrongSecretError{
			id:   t.secret_id,
			desc: "secret not for RabbitMQ connection",
		}
	}

	log.Info("using secret: ", t.secret_id)

	conn, err := amqp.Dial(rabbit_s.GetConnectionString())
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
		rabbit_s.Queue,
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
			log.Info("read msg: ", string(msg.Body))
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
	log.Info("preparing to write ", len(t.msgs), " msgs")

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

func NewRabbitMQConnection(secret_id SecretId) *rabbitMQTask {
	return &rabbitMQTask{
		secret_id: secret_id,
		qcfg: &RabbitMQQueueConfig{
			Durable:    false,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		},
	}
}

func (t *rabbitMQTask) Read() *rabbitMQReadTask {
	return &rabbitMQReadTask{
		rabbitMQTask: *t,
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

func (t *rabbitMQTask) Write(msgs [][]byte) *rabbitMQWriteTask {
	return &rabbitMQWriteTask{
		rabbitMQTask: *t,
		wcfg: &RabbitMQWriteConfig{
			Exchange:    "",
			Mandatory:   false,
			Immediate:   false,
			ContentType: "text/plain",
		},
		msgs: msgs,
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

func (t *rabbitMQReadTask) Submit() {
	BrokerPool.SubmitReadTask(t)
}

func (t *rabbitMQWriteTask) Submit() {
	BrokerPool.SubmitWriteTask(t)
}
