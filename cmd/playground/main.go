package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"mock-server/internal/util"
	"net/http"
	"time"

	zlog "github.com/rs/zerolog/log"
)

func play_brokers() {
	// broker example

	handler, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool", "test-mock-queue"))
	if err != nil {
		zlog.Error().Err(err).Msg("add new pool failed")
	}

	id := handler.NewReadTask().Schedule()
	zlog.Info().Str("id", string(id)).Msg("start reading")

	time.Sleep(1 * time.Second)

	handler, err = brokers.MPRegistry.GetMessagePool("test-pool")
	if err != nil {
		zlog.Error().Err(err).Msg("get pool failed")
	}

	handler.NewWriteTask([]string{"40", "41", "42"}).Schedule()

	time.Sleep(1 * time.Second)
}

func play_file_storage() {
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

func play_coderun() {
	var ARGS = coderun.NewDynHandleArgs([]byte(`
{
	"A": "sample_A",
	"B": 42,
	"C": ["a", "b", "c"]
}
`))

	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			zlog.Error().Err(err).Msg("borrow worker")
			return
		}

		out, err := worker.RunScript("mapper", "test.py", ARGS)
		if err != nil {
			zlog.Error().Err(err).Msg("run script")
			return
		}

		zlog.Info().Str("output", string(out)).Msg("script finished")

		worker.Return()
	}
}

func play_esb() {
	go func() {
		for x := range brokers.MPTaskScheduler.Errors() {
			fmt.Println(x)
		}
	}()

	var SCRIPT_BASIC = util.WrapCodeForEsb(`
def func(msgs):
    print(["Helllo body"])
`)
	var ARGS_BASIC = []string{}

	var SCRIPT_HARD = util.WrapCodeForEsb(`
def func(msgs):
	print(msgs[::-1])
`)
	var ARGS_HARD = []string{"msg1", "msg2", "msg3"}

	pool1, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-1", "test-mock-queue-1"))
	pool2, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-2", "test-mock-queue-2"))
	pool3, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-3", "test-mock-queue-3"))

	fs, _ := util.NewFileStorageDriver("coderun")

	if err := fs.Write("mapper", "test-mapper-basic.py", SCRIPT_BASIC); err != nil {
		zlog.Error().Err(err).Msg("write to file")
		return
	}
	if err := fs.Write("mapper", "test-mapper-hard.py", SCRIPT_HARD); err != nil {
		zlog.Error().Err(err).Msg("write to file")
		return
	}

	if err := brokers.Esb.AddEsbRecordWithMapper("test-pool-2", "test-pool-1", "test-mapper-basic.py"); err != nil {
		zlog.Error().Err(err).Msg("add esb record")
		return
	}
	if err := brokers.Esb.AddEsbRecordWithMapper("test-pool-3", "test-pool-1", "test-mapper-hard.py"); err != nil {
		zlog.Error().Err(err).Msg("add esb record")
		return
	}

	///////////////////////////////////////////////////////////////////////////////////////

	pool2.NewReadTask().Schedule()
	pool3.NewReadTask().Schedule()

	time.Sleep(2 * time.Second)

	pool2.NewWriteTask(ARGS_BASIC).Schedule()
	pool3.NewWriteTask(ARGS_HARD).Schedule()

	time.Sleep(2 * time.Second)

	pool1.NewReadTask().Schedule()
	pool1.NewReadTask().Schedule()

	time.Sleep(10 * time.Second)
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
		Str("body", string(body)).Msg("GET success")
}

func do_post(url string, content []byte) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(content))
	if err != nil {
		zlog.Error().Err(err).Str("url", url).Bytes("body", content).Msg("POST failed")
		return
	}

	var body string

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		zlog.Error().Err(err).Msg("decode failed")
		return
	}

	zlog.Info().
		Int("status", resp.StatusCode).
		Str("body", string(body)).
		Msg("POST successfully")
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
	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
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
}

func play_database() {
	endpoint := database.StaticEndpoint{
		Path:     "/test",
		Response: "Zdarova",
	}
	err := database.AddStaticEndpoint(context.TODO(), endpoint)
	if err != nil {
		panic(err)
	}
	zlog.Info().
		Str("endpoint", endpoint.Path).
		Str("response", endpoint.Response).
		Msg("Added")
	res, _ := database.ListAllStaticEndpointPaths(context.TODO())
	zlog.Info().Interface("endpoints", res).Msg("Queried")
}

func play_kafka() {
	go func() {
		for x := range brokers.MPTaskScheduler.Errors() {
			zlog.Error().Str("id", string(x.Task_id)).Err(x.Err).Msg("get error from scheduler")
		}
	}()

	handler, err := brokers.MPRegistry.AddMessagePool(brokers.NewKafkaMessagePool("test-pool-kafka", "test-topic"))
	if err != nil {
		zlog.Error().Err(err).Msg("add new pool failed")
	}

	id := handler.NewReadTask().Schedule()
	zlog.Info().Str("id", string(id)).Msg("start reading")

	time.Sleep(1 * time.Second)

	handler, err = brokers.MPRegistry.GetMessagePool("test-pool-kafka")
	if err != nil {
		zlog.Error().Err(err).Msg("get pool failed")
	}

	handler.NewWriteTask([]string{"40", "41", "42"}).Schedule()

	time.Sleep(5 * time.Second)
}

func main() {
	control.Components.Start()
	defer control.Components.Stop()

	play_brokers()
	play_file_storage()
	play_coderun()
	play_esb()
	play_server_api()
	play_database()
	play_kafka()
}
