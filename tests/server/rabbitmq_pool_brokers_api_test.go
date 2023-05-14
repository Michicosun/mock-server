package server_test

import (
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPoolBrokersRabbitmqTaskSchedulingSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()
	defer removeAllMessagePools(t)

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	rabbitmqPool := []byte(`{"pool_name":"pool","queue_name":"queue","broker":"rabbitmq"}`)

	code, body := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	// schedule write than schedule read
	messages := []string{"msg1", "msg2", "msg3"}
	writeTask := createWriteTaskBody("pool", messages)
	code, body = DoPost(poolApiEndpoint+"/write", writeTask, t)
	if code != 204 {
		t.Errorf("schedule write task failed: %s", body)
	}
	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}

	time.Sleep(1 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query write tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected to write messages be available almost simultaneously after write task request: %s", err.Error())
	}

	time.Sleep(1 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}

	// schedule read than schedule write
	moreMessages := []string{"msg4", "msg5"}
	messages = append(messages, moreMessages...)
	writeTask = createWriteTaskBody("pool", moreMessages)
	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}
	code, body = DoPost(poolApiEndpoint+"/write", writeTask, t)
	if code != 204 {
		t.Errorf("schedule write task failed: %s", body)
	}

	time.Sleep(1 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query write tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected to write messages be available almost simultaneously after write task request: %s", err.Error())
	}

	time.Sleep(1 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}
}

func TestPoolBrokersRabbitmqManyWrites(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()
	defer removeAllMessagePools(t)

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	rabbitmqPool := []byte(`{"pool_name":"pool","queue_name":"queue","broker":"rabbitmq"}`)

	code, body := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	const MESSAGE_COUNT = 100
	messages := make([]string, 0)
	for i := 0; i < MESSAGE_COUNT; i++ {
		messages = append(messages, fmt.Sprintf("msg%d", i))
	}

	// populate MESSAGE_COUNT / 2 write tasks
	for i := 0; i < MESSAGE_COUNT; i += 2 {
		writeTask := createWriteTaskBody("pool", messages[i:i+2])
		code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
		if code != 204 {
			t.Errorf("schedule write task failed: %s", body)
		}
	}
	// schedule read task
	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}

	time.Sleep(2 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query write tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
	}

	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}
}
func TestPoolBrokersRabbitmqFloodReads(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()
	defer removeAllMessagePools(t)

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	rabbitmqPool := []byte(`{"pool_name":"pool","queue_name":"queue","broker":"rabbitmq"}`)

	code, body := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	const MESSAGE_COUNT = 1000
	messages := make([]string, 0)
	for i := 0; i < MESSAGE_COUNT; i++ {
		messages = append(messages, fmt.Sprintf("msg%d", i))
	}

	// populate MESSAGE_COUNT / 4 read tasks
	for i := 0; i < MESSAGE_COUNT/4; i++ {
		code, body := DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
		if code != 204 {
			t.Errorf("schedule read task failed: %s", body)
		}
	}

	// populate MESSAGE_COUNT write tasks
	for i := 0; i < MESSAGE_COUNT; i++ {
		writeTask := createWriteTaskBody("pool", messages[i:i+1])
		code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
		if code != 204 {
			t.Errorf("schedule write task failed: %s", body)
		}
	}

	// populate MESSAGE_COUNT / 4 read tasks
	for i := 0; i < MESSAGE_COUNT/4; i++ {
		code, body := DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
		if code != 204 {
			t.Errorf("schedule read task failed: %s", body)
		}
	}

	time.Sleep(5 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query write tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
	}

	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}
}

func TestPoolBrokersRabbitmqManyPools(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()
	defer removeAllMessagePools(t)

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	const POOL_COUNT = 2
	const MESSAGE_COUNT_PER_POOL = 10
	var wg sync.WaitGroup
	wg.Add(POOL_COUNT)
	for poolNum := 0; poolNum < POOL_COUNT; poolNum++ {
		poolName := strconv.Itoa(poolNum)
		rabbitmqPool := []byte(fmt.Sprintf(`{
			"pool_name": "pool%s",
			"queue_name": "queue%s",
			"broker":"rabbitmq"
		}`, poolName, poolName))

		go func() {
			code, body := DoPost(poolApiEndpoint, rabbitmqPool, t)
			if code != 200 {
				t.Errorf("create pool failed: %s", body)
			}

			messages := make([]string, 0)
			for i := 0; i < MESSAGE_COUNT_PER_POOL; i++ {
				messages = append(messages, fmt.Sprintf("msg%d", i))
			}

			// populate MESSAGE_COUNT write tasks
			for i := 0; i < MESSAGE_COUNT_PER_POOL; i++ {
				writeTask := createWriteTaskBody("pool"+poolName, messages[i:i+1])
				code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
				if code != 204 {
					t.Errorf("schedule write task failed: %s", body)
				}
			}

			// schedule some read tasks
			for i := 0; i < 10; i++ {
				code, body := DoPost(poolApiEndpoint+"/read?pool=pool"+poolName, []byte{}, t)
				if code != 204 {
					t.Errorf("schedule read task failed: %s", body)
				}
			}

			time.Sleep(15 * time.Second)

			code, body = DoGet(poolApiEndpoint+"/write?pool=pool"+poolName, t)
			if code != 200 {
				t.Errorf("Failed to query write tasks: %s", body)
			}
			if err := compareRequestMessagesResponse(messages, body); err != nil {
				t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
			}

			code, body = DoGet(poolApiEndpoint+"/read?pool=pool"+poolName, t)
			if code != 200 {
				t.Errorf("Failed to query read tasks: %s", body)
			}
			if err := compareRequestMessagesResponse(messages, body); err != nil {
				t.Errorf("Expected completed read task after some time: %s", err.Error())
			}

			wg.Done()
		}()
	}

	wg.Wait()
}
