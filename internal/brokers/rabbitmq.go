package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/database"

	amqp "github.com/rabbitmq/amqp091-go"
	zlog "github.com/rs/zerolog/log"
)

type RabbitMQQueueConfig struct {
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Exclusive  bool                   `json:"exclusive"`
	NoWait     bool                   `json:"no_wait"`
	Args       map[string]interface{} `json:"args"`
}

type RabbitMQReadConfig struct {
	Consumer  string                 `json:"consumer"`
	AutoAck   bool                   `json:"auto_ack"`
	Exclusive bool                   `json:"exclusive"`
	NoLocal   bool                   `json:"no_local"`
	NoWait    bool                   `json:"no_wait"`
	Args      map[string]interface{} `json:"args"`
}

type RabbitMQWriteConfig struct {
	Exchange    string `json:"exchange"`
	Mandatory   bool   `json:"mandatory"`
	Immediate   bool   `json:"immediate"`
	ContentType string `json:"content_type"`
}

type RabbitMQPoolConfig struct {
	Queue string              `json:"queue"`
	Qcfg  RabbitMQQueueConfig `json:"qcfg"`
	Rcfg  RabbitMQReadConfig  `json:"rcfg"`
	Wcfg  RabbitMQWriteConfig `json:"wcfg"`
}

type RabbitMQMessagePool struct {
	name  string
	queue string
	qcfg  *RabbitMQQueueConfig
	rcfg  *RabbitMQReadConfig
	wcfg  *RabbitMQWriteConfig
}

func (mp *RabbitMQMessagePool) GetName() string {
	return mp.name
}

func (mp *RabbitMQMessagePool) GetQueue() string {
	return mp.queue
}

func (mp *RabbitMQMessagePool) GetBroker() string {
	return "rabbitmq"
}
func (mp *RabbitMQMessagePool) GetConfig() interface{} {
	return &RabbitMQPoolConfig{
		Queue: mp.queue,
		Qcfg:  *mp.qcfg,
		Rcfg:  *mp.rcfg,
		Wcfg:  *mp.wcfg,
	}
}

func (mp *RabbitMQMessagePool) GetJSONConfig() ([]byte, error) {
	return json.Marshal(mp.GetConfig())
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
	return TaskId(fmt.Sprintf("rabbitmq:%s:%s", t.pool.GetName(), t.pool.queue))
}

func (t *rabbitMQTask) getConnectionString(s *configs.RabbitMQConnectionConfig) string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", s.Username, s.Password, s.Host, s.Port)
}

func (t *rabbitMQTask) connectAndPrepare(context.Context) error {
	zlog.Info().Str("task", string(t.getTaskId())).Msg("setting up connection to rabbitmq")

	s, err := configs.GetRabbitMQConnectionConfig()
	if err != nil {
		return err
	}

	conn, err := amqp.Dial(t.getConnectionString(s))
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

			if err = database.AddTaskMessage(context.TODO(), database.TaskMessage{
				TaskId:  string(t.getTaskId()),
				Message: string(msg.Body),
			}); err != nil {
				zlog.Err(err).Msg(fmt.Sprintf("Failed to upload message for task: %s", t.getTaskId()))
				return err
			}
		}
	}
}

func (t *rabbitMQReadTask) messages() ([]string, error) {
	msgs := make([]string, 0)

	for _, msg := range t.msgs {
		msgs = append(msgs, string(msg.Body))
	}

	return msgs, nil
}

// RabbitMQ write task
type rabbitMQWriteTask struct {
	rabbitMQTask
	msgs []string
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
		if err := t.ch.PublishWithContext(ctx,
			t.pool.wcfg.Exchange,
			t.q.Name,
			t.pool.wcfg.Mandatory,
			t.pool.wcfg.Immediate,
			amqp.Publishing{
				ContentType: t.pool.wcfg.ContentType,
				Body:        []byte(msg),
			},
		); err != nil {
			return err
		}
		if err := database.AddTaskMessage(context.TODO(), database.TaskMessage{
			TaskId:  string(t.getTaskId()),
			Message: msg,
		}); err != nil {
			zlog.Err(err).Msg(fmt.Sprintf("Failed to upload message for task: %s", t.getTaskId()))
			return err
		}
	}

	return nil
}

func (t *rabbitMQWriteTask) messages() []string {
	return t.msgs
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

func createRabbitMQPoolFromDatabase(pool database.MessagePool) (*RabbitMQMessagePool, error) {
	var config RabbitMQPoolConfig
	err := json.Unmarshal([]byte(pool.Config), &config)
	if err != nil {
		return nil, err
	}
	newPool := NewRabbitMQMessagePool(pool.Name, config.Queue)
	newPool.qcfg = &config.Qcfg
	newPool.wcfg = &config.Wcfg
	newPool.rcfg = &config.Rcfg
	return newPool, nil
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

func (mp *RabbitMQMessagePool) NewReadTask() qReadTask {
	return &rabbitMQReadTask{
		rabbitMQTask: newRabbitMQBaseTask(mp),
	}
}

func (mp *RabbitMQMessagePool) NewWriteTask(data []string) qWriteTask {
	return &rabbitMQWriteTask{
		rabbitMQTask: newRabbitMQBaseTask(mp),
		msgs:         data,
	}
}
