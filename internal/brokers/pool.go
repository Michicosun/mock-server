package brokers

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"mock-server/internal/configs"
	"mock-server/internal/util"

	zlog "github.com/rs/zerolog/log"
)

var BrokerPool = &bPool{}

const (
	MAX_TIME_SHIFT = 1024
	MAX_ERRORS     = 128
)

type QueueId string

type qTask interface {
	connect_and_prepare() error
	queue_id() QueueId
	close()
}

type qReadTask interface {
	qTask
	read(ctx context.Context) error
	json() ([]byte, error)
}

type qWriteTask interface {
	qTask
	write(ctx context.Context) error
}

type QueueError struct {
	queue_id QueueId
	err      error
}

type bPool struct {
	constructor sync.Once
	cfg         *configs.PoolConfig
	read_tasks  util.BlockingQueue[qReadTask]
	write_tasks util.BlockingQueue[qWriteTask]
	errors      chan QueueError
	ctx         context.Context
	wg          sync.WaitGroup
	running_ids util.SyncSet[QueueId]
}

func (p *bPool) Init(ctx context.Context, cfg *configs.PoolConfig) {
	p.constructor.Do(func() {
		p.cfg = cfg
		p.read_tasks = util.NewUnboundedBlockingQueue[qReadTask]()
		p.write_tasks = util.NewUnboundedBlockingQueue[qWriteTask]()
		p.errors = make(chan QueueError, 100)
		p.ctx = ctx
		p.running_ids = util.NewSyncSet[QueueId]()
	})
}

func (p *bPool) Stop() {
	zlog.Info().Msg("pool: stop called")
	p.read_tasks.Close(true)
	p.write_tasks.Close(true)
	close(p.errors)
	p.wg.Wait()
	zlog.Info().Msg("all workers joined")
}

func (p *bPool) Start() {
	for i := uint32(0); i < p.cfg.R_workers; i += 1 {
		p.wg.Add(1)
		go p.r_worker_routine()
	}
	for i := uint32(0); i < p.cfg.W_workers; i += 1 {
		p.wg.Add(1)
		go p.w_worker_routine()
	}
}

func (p *bPool) Errors() <-chan QueueError {
	return p.errors
}

func (p *bPool) StopEventually(id QueueId) {
	zlog.Info().Str("task", string(id)).Msg("stopped rescheduling")
	p.running_ids.Remove(id)
}

func (p *bPool) submitReadTask(task qReadTask) QueueId {
	p.running_ids.Add(task.queue_id())
	p.read_tasks.Put(task)
	return task.queue_id()
}

func (p *bPool) submitWriteTask(task qWriteTask) QueueId {
	p.write_tasks.Put(task)
	return task.queue_id()
}

func (p *bPool) add_error(id QueueId, err error) {
	zlog.Error().Str("task", string(id)).Err(err).Msg("task failed")
	qerr := QueueError{
		queue_id: id,
		err:      err,
	}
	select {
	case p.errors <- qerr:
		zlog.Info().Str("task", string(id)).Msg("error submited")
	default:
		zlog.Info().Str("task", string(id)).Msg("failed to submit error")
	}
}

func qread(ctx context.Context, task qReadTask) error {
	zlog.Info().Str("task", string(task.queue_id())).Msg("started")
	if err := task.connect_and_prepare(); err != nil {
		return err
	}

	defer task.close()

	if err := task.read(ctx); err != nil {
		return err
	}

	// push to esb
	// write to db

	zlog.Info().Str("task", string(task.queue_id())).Err(ctx.Err()).Msg("finished")
	return nil
}

func (p *bPool) r_worker_routine() {
	for {
		elem := p.read_tasks.Get()
		if elem.IsNone() {
			zlog.Debug().Msg("r_worker Done")
			p.wg.Done()
			return
		}
		task := elem.Unwrap()
		task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Read_timeout)
		if err := qread(task_ctx, task); err != nil {
			p.add_error(task.queue_id(), err)
		}
		cancel()

		if !p.running_ids.Contains(task.queue_id()) {
			continue
		}

		go func() {
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(p.cfg.Disable_task + time.Duration(rand.Intn(MAX_TIME_SHIFT)) + time.Millisecond):
				zlog.Info().Str("task", string(task.queue_id())).Msg("rescheduled")
				p.read_tasks.Put(task)
			}
		}()
	}
}

func qwrite(ctx context.Context, task qWriteTask) error {
	zlog.Info().Str("task", string(task.queue_id())).Msg("started")
	if err := task.connect_and_prepare(); err != nil {
		return err
	}

	defer task.close()

	if err := task.write(ctx); err != nil {
		return err
	}

	// write to db
	zlog.Info().Str("task", string(task.queue_id())).Err(ctx.Err()).Msg("finished")
	return nil
}

func (p *bPool) w_worker_routine() {
	for {
		elem := p.write_tasks.Get()
		if elem.IsNone() {
			zlog.Debug().Msg("w_worker Done")
			p.wg.Done()
			return
		}
		task := elem.Unwrap()
		task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Write_timeout)
		if err := qwrite(task_ctx, task); err != nil {
			p.add_error(task.queue_id(), err)
		}
		cancel()
	}
}
