package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/coderun/docker-provider"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/server"
	"mock-server/internal/util"
	"net/http"
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

func do_get(url string) {
	resp, err := http.Get(url)
	if err != nil {
		zlog.Error().Err(err).Str("url", url).Msg("GET failed")
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to read body")
		return
	}

	zlog.Info().
		Int("status", resp.StatusCode).
		Str("body", string(body)).
		Msg("GET success")
}

func do_post(url string, content []byte) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(content))
	if err != nil {
		zlog.Error().Err(err).Str("url", url).Bytes("body", content).Msg("POST failed")
		return
	}

	var res map[string]string

	json.NewDecoder(resp.Body).Decode(&res)

	zlog.Info().
		Int("status", resp.StatusCode).
		Msg("POST success")
}

func do_delete(url string) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		zlog.Error().Err(err).Str("url", url).Msg("DELETE failed")
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		zlog.Error().Err(err).Str("url", url).Msg("DELETE failed")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to read body")
		return
	}

	zlog.Info().
		Int("status", resp.StatusCode).
		Str("body", string(body)).
		Msg("DELETE success")
}

func play_server_api() {
	server.Server.Init(configs.GetServerConfig())

	go func() {
		time.Sleep(1 * time.Second)

		cfg := configs.GetServerConfig()
		endpoint := fmt.Sprintf("http://%s:%s", cfg.Addr, cfg.Port)
		staticApiEndpoint := endpoint + "/api/routes/static"

		{
			url := endpoint + "/api/ping"
			do_get(url)
		}

		{
			testUrl := endpoint + "/test_url"

			// no routes created -> 404
			do_get(testUrl)
			// expects []
			do_get(staticApiEndpoint)

			// create route /test_url with reponse `hello`
			requestBody := []byte(`{
                "path": "/test_url",
                "expected_response": "hello"
            }`)
			do_post(staticApiEndpoint, requestBody)

			// expects `hello`
			do_get(testUrl)
			// expects ["/test_url"]
			do_get(staticApiEndpoint)

			// detele /test_url
			do_delete(staticApiEndpoint + "?path=/test_url")

			// /test_url deleted -> 404
			do_get(testUrl)
			// expects []
			do_get(staticApiEndpoint)
		}
	}()

	server.Server.Start()
}

func main() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	// create root context
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	zlog.Info().Msg("starting...")

	// play_brokers(ctx, cancel)
	// play_docker(ctx, cancel)
	// play_file_storage(ctx, cancel)
	// play_coderun(ctx, cancel)
	// play_esb(ctx, cancel)
	play_server_api()
}
