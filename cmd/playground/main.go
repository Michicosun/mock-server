package main

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/coderun/docker-provider"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/util"
	"time"

	zlog "github.com/rs/zerolog/log"
)

func play_brokers(ctx context.Context, cancel context.CancelFunc) {
	// broker example

	brokers.MPRegistry.Init()

	brokers.MPTaskScheduler.Init(ctx, configs.GetMPTaskSchedulerConfig())
	brokers.MPTaskScheduler.Start()

	handler, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool", "test-mock-queue"))
	if err != nil {
		zlog.Error().Err(err).Msg("add new pool failed")
	}

	id := handler.NewReadTask().Schedule()
	zlog.Info().Str("id", string(id)).Msg("start reading")

	<-time.After(1 * time.Second)

	handler, err = brokers.MPRegistry.GetMessagePool("test-pool")
	if err != nil {
		zlog.Error().Err(err).Msg("get pool failed")
	}

	handler.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	<-time.After(1 * time.Second)

	cancel()

	brokers.MPTaskScheduler.Stop()
	for x := range brokers.MPTaskScheduler.Errors() {
		fmt.Println(x)
	}
}

func play_docker(ctx context.Context, cancel context.CancelFunc) {
	provider, err := docker.NewDockerProvider(ctx, &configs.GetCoderunConfig().WorkerConfig.Resources)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create provider")
		return
	}

	err = provider.BuildWorkerImage()
	if err != nil {
		zlog.Error().Err(err).Msg("cannot build image")
		return
	}

	id, err := provider.CreateWorkerContainer("8095")
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create container")
		return
	}

	err = provider.StartWorkerContainer(id)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create container")
		return
	}

	time.Sleep(time.Second * 100)

	err = provider.RemoveWorkerContainer(id, true)
	if err != nil {
		zlog.Error().Err(err).Msg("cannot remove container")
		return
	}
	provider.Close()
}

func play_file_storage(ctx context.Context, cancel context.CancelFunc) {
	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		zlog.Error().Err(err).Msg("cannot create filestorage")
		return
	}
	err = fs.Write("mapper", "test.py", []byte(`print("Hello, world!")`))
	if err != nil {
		zlog.Error().Err(err).Msg("write failed")
		return
	}
	s, err := fs.Read("mapper", "test.py")
	if err != nil {
		zlog.Error().Err(err).Msg("read failed")
		return
	}
	zlog.Info().Str("text", s).Msg("read file successfuly")
}

type ComplexArgs struct {
	A string   `json:"A"`
	B int      `json:"B"`
	C []string `json:"C"`
}

func play_coderun(ctx context.Context, cancel context.CancelFunc) {
	err := coderun.WorkerWatcher.Init(ctx, configs.GetCoderunConfig())

	time.Sleep(5 * time.Second)

	if err != nil {
		zlog.Error().Err(err).Msg("watcher init")
		return
	}

	for i := 0; i < 1000; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			zlog.Error().Err(err).Msg("borrow worker")
			return
		}

		out, err := worker.RunScript("mapper", "test.py", ComplexArgs{
			A: "sample_A",
			B: 42,
			C: []string{"a", "b", "c"},
		})
		if err != nil {
			zlog.Error().Err(err).Msg("run script")
			return
		}

		zlog.Info().Str("output", string(out)).Msg("script finished")

		worker.Return()
	}

	cancel()

	coderun.WorkerWatcher.Stop()
}

func play_esb(ctx context.Context, cancel context.CancelFunc) {
	brokers.MPRegistry.Init()
	brokers.Esb.Init()

	brokers.MPTaskScheduler.Init(ctx, configs.GetMPTaskSchedulerConfig())

	brokers.MPTaskScheduler.Start()
	defer brokers.MPTaskScheduler.Stop()

	coderun.WorkerWatcher.Init(ctx, configs.GetCoderunConfig())
	defer coderun.WorkerWatcher.Stop()

	pool1, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-1", "test-mock-queue-1"))
	pool2, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-2", "test-mock-queue-2"))

	fs, _ := util.NewFileStorageDriver("coderun")
	fs.Write("mapper", "test-mapper.py", []byte(`print([[72, 69, 76, 76, 79], [87, 79, 82, 76, 68], [33]])`))

	brokers.Esb.AddEsbRecordWithMapper("test-pool-1", "test-pool-2", "test-mapper.py")

	///////////////////////////////////////////////////////////////////////////////////////

	go func() {
		for x := range brokers.MPTaskScheduler.Errors() {
			fmt.Println(x)
		}
	}()

	pool1.NewReadTask().Schedule()

	<-time.After(2 * time.Second)

	pool1.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	<-time.After(2 * time.Second)

	pool2.NewReadTask().Schedule()

	<-time.After(10 * time.Second)

	cancel()
}

func main() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	// create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zlog.Info().Msg("starting...")

	// play_brokers(ctx, cancel)
	// play_docker(ctx, cancel)
	// play_file_storage(ctx, cancel)
	// play_coderun(ctx, cancel)
	play_esb(ctx, cancel)
}
