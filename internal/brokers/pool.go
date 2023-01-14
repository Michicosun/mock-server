package brokers

import (
	"context"
	"sync"
	"time"

	"mock-server/internal/configs"
	"mock-server/internal/util"

	zlog "github.com/rs/zerolog/log"
)

var BrokerPool = &bPool{}

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

type bPool struct {
	constructor sync.Once
	cfg         *configs.PoolConfig
	read_tasks  chan qReadTask
	write_tasks chan qWriteTask
	ctx         context.Context
	wg          sync.WaitGroup
	running_ids util.SyncSet[QueueId]
}

func (p *bPool) Init(ctx context.Context, cfg *configs.PoolConfig) {
	p.constructor.Do(func() {
		p.cfg = cfg
		p.read_tasks = make(chan qReadTask, cfg.R_workers*2)
		p.write_tasks = make(chan qWriteTask, cfg.W_workers*2)
		p.ctx = ctx
		p.running_ids = util.NewSyncSet[QueueId]()
	})
}

func (p *bPool) Wait() {
	zlog.Info().Msg("pool: wait called")
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

func (p *bPool) StopEventually(id QueueId) {
	zlog.Info().Msgf("stopping task: %s", id)
	p.running_ids.Remove(id)
}

func (p *bPool) submitReadTask(task qReadTask) QueueId {
	p.running_ids.Add(task.queue_id())
	p.read_tasks <- task
	return task.queue_id()
}

func (p *bPool) submitWriteTask(task qWriteTask) QueueId {
	p.write_tasks <- task
	return task.queue_id()
}

func qread(ctx context.Context, task qReadTask) error {
	zlog.Info().Msgf("starting read task: %s", task.queue_id())
	if err := task.connect_and_prepare(); err != nil {
		return err
	}

	defer task.close()

	if err := task.read(ctx); err != nil {
		return err
	}

	// push to esb
	// write to db

	zlog.Info().Msgf("completed read task: %s, ctx error: %e", task.queue_id(), ctx.Err())
	return nil
}

func (p *bPool) r_worker_routine() {
	for {
		select {
		case <-p.ctx.Done():
			zlog.Debug().Msg("r_worker Done")
			p.wg.Done()
			return
		case task := <-p.read_tasks:
			task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Read_timeout)
			if err := qread(task_ctx, task); err != nil {
				zlog.Error().Err(err).Msgf("read task: %s", task.queue_id())
			}
			cancel()

			if !p.running_ids.Contains(task.queue_id()) {
				continue
			}

			go func() {
				select {
				case <-p.ctx.Done():
					return
				case <-time.After(p.cfg.Disable_task):
					zlog.Info().Msgf("reschedule reading from queue %s", task.queue_id())
					p.read_tasks <- task
				}
			}()
		}
	}
}

func qwrite(ctx context.Context, task qWriteTask) error {
	zlog.Info().Msgf("starting write task: %s", task.queue_id())
	if err := task.connect_and_prepare(); err != nil {
		return err
	}

	defer task.close()

	if err := task.write(ctx); err != nil {
		return err
	}

	// write to db
	zlog.Info().Msgf("completed write task: %s, ctx error: %e", task.queue_id(), ctx.Err())
	return nil
}

func (p *bPool) w_worker_routine() {
	for {
		select {
		case <-p.ctx.Done():
			zlog.Debug().Msg("w_worker Done")
			p.wg.Done()
			return
		case task := <-p.write_tasks:
			task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Write_timeout)
			if err := qwrite(task_ctx, task); err != nil {
				zlog.Error().Err(err).Msgf("write task: %s", task.queue_id())
			}
			cancel()
		}
	}
}
