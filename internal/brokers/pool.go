package brokers

import (
	"context"
	"sync"
	"time"

	"mock-server/internal/configs"

	zlog "github.com/rs/zerolog/log"
)

var BrokerPool = &bPool{}

type qTask interface {
	connect_and_prepare() error
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

var onlyOnce sync.Once

func (p *bPool) Init(ctx context.Context, cfg *configs.PoolConfig) {
	onlyOnce.Do(func() {
		p.cfg = cfg
		p.read_tasks = make(chan qReadTask, cfg.R_workers*2)
		p.write_tasks = make(chan qWriteTask, cfg.W_workers*2)
		p.ctx = ctx
	})
}

type bPool struct {
	cfg         *configs.PoolConfig
	read_tasks  chan qReadTask
	write_tasks chan qWriteTask
	ctx         context.Context
	wg          sync.WaitGroup
}

func (p *bPool) SubmitReadTask(task qReadTask) {
	p.read_tasks <- task
}

func (p *bPool) SubmitWriteTask(task qWriteTask) {
	p.write_tasks <- task
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

func qread(ctx context.Context, task qReadTask) error {
	err := task.connect_and_prepare()
	if err != nil {
		return err
	}

	defer task.close()

	for {
		err := task.read(ctx)
		if ctx.Err() != nil {
			zlog.Debug().Err(ctx.Err())
			// push to esb
			// write to db
			return nil
		}
		if err != nil {
			zlog.Error().Err(err)
			return err
		}
	}
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
			err := qread(task_ctx, task)
			cancel()

			if err != nil {
				zlog.Error().Err(err)
			}

			go func() {
				select {
				case <-p.ctx.Done():
					return
				case <-time.After(p.cfg.Disable_task):
					zlog.Info().Msgf("reschedule reading from queue %v", task)
					p.read_tasks <- task
				}
			}()
		}
	}
}

func qwrite(ctx context.Context, task qWriteTask) error {
	err := task.connect_and_prepare()
	if err != nil {
		return err
	}

	defer task.close()

	err = task.write(ctx)
	if err != nil {
		zlog.Error().Err(err)
		return err
	}

	// write to db
	zlog.Info().Msgf("wrote msg, task: %v", task)
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
			err := qwrite(task_ctx, task)
			cancel()

			if err != nil {
				zlog.Error().Err(err)
			}
		}
	}
}
