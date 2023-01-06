package brokers

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
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

type BPoolConfig struct {
	R_workers     int
	W_workers     int
	Read_timeout  time.Duration
	Write_timeout time.Duration
	Disable_task  time.Duration
}

var onlyOnce sync.Once

func (p *bPool) Init(ctx context.Context, cfg BPoolConfig) {
	onlyOnce.Do(func() {
		p.cfg = cfg
		p.read_tasks = make(chan qReadTask, cfg.R_workers*2)
		p.write_tasks = make(chan qWriteTask, cfg.W_workers*2)
		p.ctx = ctx
	})
}

type bPool struct {
	cfg         BPoolConfig
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
	log.Info("pool: wait called")
	p.wg.Wait()
	log.Info("all workers joined")
}

func (p *bPool) Start() {
	for i := 0; i < p.cfg.R_workers; i += 1 {
		p.wg.Add(1)
		go p.r_worker_routine()
	}
	for i := 0; i < p.cfg.W_workers; i += 1 {
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
			log.Debug(ctx.Err())
			// push to esb
			// write to db
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}
	}
}

func (p *bPool) r_worker_routine() {
	for {
		select {
		case <-p.ctx.Done():
			log.Debug("r_worker Done")
			p.wg.Done()
			return
		case task := <-p.read_tasks:
			task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Read_timeout)
			err := qread(task_ctx, task)
			cancel()

			if err != nil {
				log.Error(err.Error())
			}

			go func() {
				select {
				case <-p.ctx.Done():
					return
				case <-time.After(p.cfg.Disable_task):
					log.Info("reschedule reading from queue", task)
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
		log.Error(err)
		return err
	}

	// write to db
	log.Info("wrote msg, task: ", task)
	return nil
}

func (p *bPool) w_worker_routine() {
	for {
		select {
		case <-p.ctx.Done():
			log.Debug("w_worker Done")
			p.wg.Done()
			return
		case task := <-p.write_tasks:
			task_ctx, cancel := context.WithTimeout(p.ctx, p.cfg.Write_timeout)
			err := qwrite(task_ctx, task)
			cancel()

			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}
