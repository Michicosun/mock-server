package brokers

import (
	"context"
	"fmt"
	"mock-server/internal/configs"

	amqp "github.com/rabbitmq/amqp091-go"
	zlog "github.com/rs/zerolog/log"
)

type RabbitMQQueueConfig struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       map[string]interface{}
}

type RabbitMQReadConfig struct {
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      map[string]interface{}
}

type RabbitMQWriteConfig struct {
	Exchange    string
	Mandatory   bool
	Immediate   bool
	ContentType string
}

type RabbitMQMessagePool struct {
	name  string
	queue string
	qcfg  *RabbitMQQueueConfig
	rcfg  *RabbitMQReadConfig
	wcfg  *RabbitMQWriteConfig
}

type rabbitMQMessagePoolHandler struct {
	pool *RabbitMQMessagePool
}

func (mp *RabbitMQMessagePool) getName() string {
	return mp.name
}

func (mp *RabbitMQMessagePool) getBroker() string {
	return "rabbitmq"
}

func (mp *RabbitMQMessagePool) getHandler() MessagePoolHandler {
	return &rabbitMQMessagePoolHandler{
		pool: mp,
	}
}

// RabbitMQ base task
type rabbitMQTask struct {
	pool *RabbitMQMessagePool
	conn *amqp.Connection
	ch   *amqp.Channel
	q    *amqp.Queue
}

func (t *rabbitMQTask) getMessagePool() MessagePool {
	return t.pool
}

func (t *rabbitMQTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("rabbitmq:%s:%s", t.pool.getName(), t.pool.queue))
}

func getConnectionString(s *configs.RabbitMQConnectionConfig) string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", s.Username, s.Password, s.Host, s.Port)
}

func (t *rabbitMQTask) connectAndPrepare() error {
	zlog.Info().Str("task", string(t.getTaskId())).Msg("setting up connection to rabbitmq")

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
		t.pool.queue,
		t.pool.qcfg.Durable,
		t.pool.qcfg.AutoDelete,
		t.pool.qcfg.Exclusive,
		t.pool.qcfg.NoWait,
		t.pool.qcfg.Args,
	)
	if err != nil {
		return err
	}
	t.q = &q

	zlog.Info().Str("task", string(t.getTaskId())).Msg("connection established")
	return nil
}

func (t *rabbitMQTask) close() {
	t.ch.Close()
	t.conn.Close()
}

// RabbitMQ read task
type rabbitMQReadTask struct {
	rabbitMQTask
	msgs []amqp.Delivery
}

func (t *rabbitMQReadTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("%s:read", t.rabbitMQTask.getTaskId()))
}

func (t *rabbitMQReadTask) Schedule() TaskId {
	return MPTaskScheduler.submitReadTask(t)
}

func (t *rabbitMQReadTask) read(ctx context.Context) error {
	msgs, err := t.ch.Consume(
		t.q.Name,
		t.pool.rcfg.Consumer,
		t.pool.rcfg.AutoAck,
		t.pool.rcfg.Exclusive,
		t.pool.rcfg.NoLocal,
		t.pool.rcfg.NoWait,
		t.pool.rcfg.Args,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgs:
			zlog.Debug().Str("task", string(t.getTaskId())).Bytes("msg", msg.Body).Msg("read msg")
			t.msgs = append(t.msgs, msg)
		}
	}
}

func (t *rabbitMQReadTask) json() ([][]byte, error) {
	msgs := make([][]byte, 0)

	for _, msg := range t.msgs {
		msgs = append(msgs, msg.Body)
	}

	return msgs, nil
}

// RabbitMQ write task
type rabbitMQWriteTask struct {
	rabbitMQTask
	msgs [][]byte
}

func (t *rabbitMQWriteTask) getTaskId() TaskId {
	return TaskId(fmt.Sprintf("%s:write", t.rabbitMQTask.getTaskId()))
}

func (t *rabbitMQWriteTask) Schedule() TaskId {
	return MPTaskScheduler.submitWriteTask(t)
}

func (t *rabbitMQWriteTask) write(ctx context.Context) error {
	zlog.Info().Str("task", string(t.getTaskId())).Int("msgs_cnt", len(t.msgs)).Msg("preparing to write")

	for _, msg := range t.msgs {
		err := t.ch.PublishWithContext(ctx,
			t.pool.wcfg.Exchange,
			t.q.Name,
			t.pool.wcfg.Mandatory,
			t.pool.wcfg.Immediate,
			amqp.Publishing{
				ContentType: t.pool.wcfg.ContentType,
				Body:        msg,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewRabbitMQMessagePool(name string, queue string) *RabbitMQMessagePool {
	return &RabbitMQMessagePool{
		name:  name,
		queue: queue,
		qcfg: &RabbitMQQueueConfig{
			Durable:    false,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		},
		rcfg: &RabbitMQReadConfig{
			Consumer:  "",
			AutoAck:   true,
			Exclusive: false,
			NoLocal:   false,
			NoWait:    false,
			Args:      nil,
		},
		wcfg: &RabbitMQWriteConfig{
			Exchange:    "",
			Mandatory:   false,
			Immediate:   false,
			ContentType: "text/plain",
		},
	}
}

func (mp *RabbitMQMessagePool) SetQueueConfig(cfg RabbitMQQueueConfig) *RabbitMQMessagePool {
	mp.qcfg = &cfg
	return mp
}

func (mp *RabbitMQMessagePool) SetReadConfig(cfg RabbitMQReadConfig) *RabbitMQMessagePool {
	mp.rcfg = &cfg
	return mp
}

func (mp *RabbitMQMessagePool) SetWriteConfig(cfg RabbitMQWriteConfig) *RabbitMQMessagePool {
	mp.wcfg = &cfg
	return mp
}

func newRabbitMQBaseTask(pool *RabbitMQMessagePool) rabbitMQTask {
	return rabbitMQTask{
		pool: pool,
	}
}

func (h *rabbitMQMessagePoolHandler) NewReadTask() qReadTask {
	return &rabbitMQReadTask{
		rabbitMQTask: newRabbitMQBaseTask(h.pool),
	}
}

func (h *rabbitMQMessagePoolHandler) NewWriteTask(data [][]byte) qWriteTask {
	return &rabbitMQWriteTask{
		rabbitMQTask: newRabbitMQBaseTask(h.pool),
		msgs:         data,
	}
}
