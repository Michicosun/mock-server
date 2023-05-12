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

func TestPoolsBrokersKamikadze(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	const POOL_COUNT = 4
	const MESSAGE_COUNT_PER_POOL = 20
	var wg sync.WaitGroup
	wg.Add(POOL_COUNT)
	for poolNum := 0; poolNum < POOL_COUNT; poolNum++ {
		poolName := strconv.Itoa(poolNum)
		if poolNum < POOL_COUNT/2 {
			rabbitmqPool := []byte(fmt.Sprintf(`{
			"pool_name": "pool%s",
			"queue_name": "queue%s",
			"broker":"rabbitmq"
		}`, poolName, poolName))

			code, body := DoPost(poolApiEndpoint, rabbitmqPool, t)
			if code != 200 {
				t.Errorf("create pool failed: %s", body)
			}

		} else {
			kafkaPool := []byte(fmt.Sprintf(`{
			"pool_name": "pool%s",
			"topic_name": "queue%s",
			"broker":"kafka"
		}`, poolName, poolName))

			code, body := DoPost(poolApiEndpoint, kafkaPool, t)
			if code != 200 {
				t.Errorf("create pool failed: %s", body)
			}

		}

		go func() {
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

			code, body := DoGet(poolApiEndpoint+"/write?pool=pool"+poolName, t)
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