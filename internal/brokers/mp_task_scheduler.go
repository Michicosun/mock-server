package brokers

import (
	"context"
	"sync"

	"mock-server/internal/configs"
	"mock-server/internal/util"

	zlog "github.com/rs/zerolog/log"
)

var MPTaskScheduler = &mpTaskScheduler{}

const (
	MAX_ERRORS = 128
)

type TaskId string

type qTask interface {
	connectAndPrepare(ctx context.Context) error
	getTaskId() TaskId
	getMessagePool() MessagePool
	close()
}

type qReadTask interface {
	qTask
	Schedule() TaskId
	read(ctx context.Context) error
}

type qWriteTask interface {
	qTask
	Schedule() TaskId
	write(ctx context.Context) error
}

type TaskError struct {
	Task_id TaskId
	Err     error
}

type mpTaskScheduler struct {
	cfg                *configs.MPTaskSchedulerConfig
	read_tasks         util.BlockingQueue[qReadTask]
	write_tasks        util.BlockingQueue[qWriteTask]
	running_read_tasks util.SyncSet[TaskId]
	errors             chan TaskError
	ctx                context.Context
	wg                 sync.WaitGroup
}

func (mps *mpTaskScheduler) Init(ctx context.Context, cfg *configs.MPTaskSchedulerConfig) {
	mps.cfg = cfg
	mps.read_tasks = util.NewUnboundedBlockingQueue[qReadTask]()
	mps.write_tasks = util.NewUnboundedBlockingQueue[qWriteTask]()
	mps.running_read_tasks = util.NewSyncSet[TaskId]()
	mps.errors = make(chan TaskError, MAX_ERRORS)
	mps.ctx = ctx
}

func (mps *mpTaskScheduler) Stop() {
	zlog.Info().Msg("stop called")
	mps.read_tasks.Close(true)
	mps.write_tasks.Close(true)
	close(mps.errors)
	mps.wg.Wait()
	zlog.Info().Msg("all workers joined")
}

func (mps *mpTaskScheduler) Start() {
	zlog.Info().Msg("starting broker task scheduler")
	for i := uint32(0); i < mps.cfg.R_workers; i += 1 {
		mps.wg.Add(1)
		go mps.rWorkerRoutine()
	}
	for i := uint32(0); i < mps.cfg.W_workers; i += 1 {
		mps.wg.Add(1)
		go mps.wWorkerRoutine()
	}
}

func (mps *mpTaskScheduler) Errors() <-chan TaskError {
	return mps.errors
}

func (mps *mpTaskScheduler) submitReadTask(task qReadTask) TaskId {
	if mps.running_read_tasks.Insert(task.getTaskId()) {
		mps.read_tasks.Put(task)
	}
	return task.getTaskId()
}

func (mps *mpTaskScheduler) submitWriteTask(task qWriteTask) TaskId {
	mps.write_tasks.Put(task)
	return task.getTaskId()
}

func (mps *mpTaskScheduler) submitError(id TaskId, err error) {
	zlog.Error().Str("task", string(id)).Err(err).Msg("task failed")
	qerr := TaskError{
		Task_id: id,
		Err:     err,
	}
	select {
	case mps.errors <- qerr:
		zlog.Info().Str("task", string(id)).Msg("error submited")
	default:
		zlog.Info().Str("task", string(id)).Msg("failed to submit error")
	}
}

func qread(ctx context.Context, task qReadTask) error {
	zlog.Info().Str("task", string(task.getTaskId())).Msg("started")
	if err := task.connectAndPrepare(ctx); err != nil {
		return err
	}

	defer task.close()

	if err := task.read(ctx); err != nil {
		return err
	}

	zlog.Info().Str("task", string(task.getTaskId())).Err(ctx.Err()).Msg("finished")
	return nil
}

func (mps *mpTaskScheduler) rWorkerRoutine() {
	for {
		elem := mps.read_tasks.Get()
		if elem.IsNone() {
			zlog.Debug().Msg("r_worker Done")
			mps.wg.Done()
			return
		}
		task := elem.Unwrap()
		task_ctx, cancel := context.WithTimeout(mps.ctx, mps.cfg.Read_timeout)
		if err := qread(task_ctx, task); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			mps.submitError(task.getTaskId(), err)
		}
		mps.running_read_tasks.Remove(task.getTaskId())
		cancel()
	}
}

func qwrite(ctx context.Context, task qWriteTask) error {
	zlog.Info().Str("task", string(task.getTaskId())).Msg("started")
	if err := task.connectAndPrepare(ctx); err != nil {
		return err
	}

	defer task.close()

	if err := task.write(ctx); err != nil {
		return err
	}

	zlog.Info().Str("task", string(task.getTaskId())).Err(ctx.Err()).Msg("finished")
	return nil
}

func (mps *mpTaskScheduler) wWorkerRoutine() {
	for {
		elem := mps.write_tasks.Get()
		if elem.IsNone() {
			zlog.Debug().Msg("w_worker Done")
			mps.wg.Done()
			return
		}
		task := elem.Unwrap()
		task_ctx, cancel := context.WithTimeout(mps.ctx, mps.cfg.Write_timeout)
		if err := qwrite(task_ctx, task); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			mps.submitError(task.getTaskId(), err)
		}
		cancel()
	}
}
