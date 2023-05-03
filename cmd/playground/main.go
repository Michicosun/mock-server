package main

import (
	"bytes"
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

	handler.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	time.Sleep(1 * time.Second)
}

func play_file_storage() {
	fmt.Println("play_file_storage")
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

func play_coderun() {
	for i := 0; i < 10; i += 1 {
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
}

func play_esb() {
	pool1, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-1", "test-mock-queue-1"))
	pool2, _ := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-2", "test-mock-queue-2"))

	fs, _ := util.NewFileStorageDriver("coderun")
	err := fs.Write("mapper", "test-mapper.py", []byte(`print([[72, 69, 76, 76, 79], [87, 79, 82, 76, 68], [33]])`))
	if err != nil {
		zlog.Error().Err(err).Msg("write to file")
		return
	}

	err = brokers.Esb.AddEsbRecordWithMapper("test-pool-1", "test-pool-2", "test-mapper.py")
	if err != nil {
		zlog.Error().Err(err).Msg("add esb record")
		return
	}

	///////////////////////////////////////////////////////////////////////////////////////

	go func() {
		for x := range brokers.MPTaskScheduler.Errors() {
			fmt.Println(x)
		}
	}()

	pool1.NewReadTask().Schedule()

	time.Sleep(2 * time.Second)

	pool1.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	time.Sleep(2 * time.Second)

	pool2.NewReadTask().Schedule()

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
		Msg(fmt.Sprintf("POST success with message: %s", body))
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
	database.AddStaticEndpoint(endpoint)
	fmt.Printf("Add endpoint %s\n", endpoint)
	res, _ := database.ListAllStaticEndpoints()
	fmt.Println("Found endpoints:")
	for _, endpoint := range res {
		fmt.Println(endpoint)
	}
}

func main() {
	control.Components.Start()
	defer control.Components.Stop()

	// play_brokers()
	// play_file_storage()
	// play_coderun()
	// play_esb()
	// play_server_api()
	play_database()
}
