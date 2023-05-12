package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/server/protocol"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPoolBrokersSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	// expects []
	code, body := DoGet(poolApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all pools")
	}

	if !bytes.Equal(body, []byte(`{"pools":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"pools":[]}`, body)
	}

	// create rabbitmq pool
	rabbitmqPool := []byte(`{"pool_name":"rabbitmq_pool","queue_name":"rabbitmq_queue","broker":"rabbitmq"}`)
	code, _ = DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	// expects to get created pool
	code, body = DoGet(poolApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all pools")
	}

	expectedPoolsList := fmt.Sprintf(`{"pools":[%s]}`, rabbitmqPool)
	if !bytes.Equal(body, []byte(expectedPoolsList)) {
		t.Errorf(`list request must contain created pool: %s != %s`, body, expectedPoolsList)
	}

	// create kafka pool
	kafkaPool := []byte(`{"pool_name":"kafka_pool","topic_name":"kafka_queue","broker":"kafka"}`)
	code, _ = DoPost(poolApiEndpoint, kafkaPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	// expects to get both created pools
	code, body = DoGet(poolApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all pools")
	}

	expectedPoolsList = fmt.Sprintf(`{"pools":[%s,%s]}`, rabbitmqPool, kafkaPool)
	if !bytes.Equal(body, []byte(expectedPoolsList)) {
		t.Errorf(`list request must contain both created pools: %s != %s`, body, expectedPoolsList)
	}

	// query first pool config
	code, body = DoGet(poolApiEndpoint+"/config?pool=rabbitmq_pool", t)
	if code != 200 {
		t.Errorf("expected 200 code response on get pool config")
	}
	t.Logf("config: %s", body)

	// remove second pool
	code = DoDelete(poolApiEndpoint+"?pool=kafka_pool", t)
	if code != 204 {
		t.Errorf("expected 204 code response on remove existing pool request")
	}

	// expects to get only first pool
	code, body = DoGet(poolApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all pools")
	}

	expectedPoolsList = fmt.Sprintf(`{"pools":[%s]}`, rabbitmqPool)
	if !bytes.Equal(body, []byte(expectedPoolsList)) {
		t.Errorf(`list request must contain both created pools: %s != %s`, body, expectedPoolsList)
	}
}

func TestPoolBrokersBadQueryBodies(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	var badBodies = map[string]string{
		"empty": `{}`,
		"no pool name": `{
			"queue_name": "t"
			"broker": "rabbitmq"
		}`,
		"no pool broker": `{
			"pool_name": "t",
			"queue_name": "t"
		}`,
		"bad broker": `{
			"pool_name": "t",
			"queue_name": "t",
			"broker": "akakf"
		}`,
		"no queue name for rabbitmq broker": `{
			"pool_name": "t",
			"topic_name": "t",
			"broker": "rabbitmq"
		}`,
		"no topic name for kafka broker": `{
			"pool_name": "t",
			"queue_name": "t",
			"broker": "kafka"
		}`,
	}

	for expectedFailureTrigger, badBody := range badBodies {
		code, _ := DoPost(poolApiEndpoint, []byte(badBody), t)
		if code != 400 {
			t.Errorf("expected to fail request because of %s body", expectedFailureTrigger)
		}
	}

}

func TestPoolBrokersDoublePost(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	rabbitmqPool := []byte(`{"pool_name":"pool","queue_name":"queue","broker":"rabbitmq"}`)
	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

	code, _ := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	code, _ = DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 409 {
		t.Errorf("expected to be impossible to create pool with the same name")
	}

	code, _ = DoPost(poolApiEndpoint, kafkaPool, t)
	if code != 409 {
		t.Errorf("expected to be impossible to create pool with the same name even from another broker")
	}
}

func TestPoolBrokersRabbitmqTaskSchedulingSimple(t *testing.T) {
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

	rabbitmqPool := []byte(`{"pool_name":"pool","queue_name":"queue","broker":"rabbitmq"}`)

	code, _ := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	// schedule write than schedule read
	messages := []string{"msg1", "msg2", "msg3"}
	writeTask := createWriteTaskBody("pool", messages)
	code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
	if code != 204 {
		t.Errorf("schedule write task failed: %s", body)
	}
	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
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
		t.Errorf("Failed to query write tasks: %s", body)
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
		t.Errorf("schedule write task failed: %s", body)
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

	code, _ := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	const MESSAGE_COUNT = 5000
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
	code, body := DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
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

	code, _ := DoPost(poolApiEndpoint, rabbitmqPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
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

	code, body := DoGet(poolApiEndpoint+"/write?pool=pool", t)
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

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	const POOL_COUNT = 10
	const MESSAGE_COUNT_PER_POOL = 300
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
			code, _ := DoPost(poolApiEndpoint, rabbitmqPool, t)
			if code != 200 {
				t.Errorf("create pool failed")
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

func TestPoolBrokersKafkaTaskSchedulingSimple(t *testing.T) {
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

	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

	code, _ := DoPost(poolApiEndpoint, kafkaPool, t)
	if code != 200 {
		t.Errorf("create pool failed")
	}

	// schedule write than schedule read
	messages := []string{"msg1", "msg2", "msg3"}
	writeTask := createWriteTaskBody("pool", messages)
	code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
	if code != 204 {
		t.Errorf("schedule write task failed: %s", body)
	}
	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}

	time.Sleep(10 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
	if code != 200 {
		t.Errorf("Failed to query write tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected to write messages be available almost simultaneously after write task request: %s", err.Error())
	}

	time.Sleep(3 * time.Second)

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

	time.Sleep(10 * time.Second)

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

func TestPoolBrokersKafkaManyWrites(t *testing.T) {
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

	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

	code, body := DoPost(poolApiEndpoint, kafkaPool, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	const MESSAGE_COUNT = 5000
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
		t.Errorf("schedule write task failed: %s", body)
	}

	time.Sleep(10 * time.Second)

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

func TestPoolBrokersKafkaFloodReads(t *testing.T) {
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

	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

	code, body := DoPost(poolApiEndpoint, kafkaPool, t)
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

	time.Sleep(20 * time.Second)

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

func TestPoolBrokersKafkaManyPools(t *testing.T) {
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

	const POOL_COUNT = 10
	const MESSAGE_COUNT_PER_POOL = 200
	var wg sync.WaitGroup
	wg.Add(POOL_COUNT)
	for poolNum := 0; poolNum < POOL_COUNT; poolNum++ {
		poolName := strconv.Itoa(poolNum)
		kafkaPool := []byte(fmt.Sprintf(`{
			"pool_name": "pool%s",
			"topic_name": "queue%s",
			"broker":"kafka"
		}`, poolName, poolName))

		go func() {
			code, body := DoPost(poolApiEndpoint, kafkaPool, t)
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

			time.Sleep(30 * time.Second)

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

	const POOL_COUNT = 10
	const MESSAGE_COUNT_PER_POOL = 300
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

func createWriteTaskBody(poolName string, messages []string) []byte {
	brokerTask := protocol.BrokerTask{
		PoolName: poolName,
		Messages: messages,
	}
	body, _ := json.Marshal(brokerTask)

	return body
}

type testMessages struct {
	Messages []string `json:"messages"`
}

func compareRequestMessagesResponse(expected []string, actualBody []byte) error {
	var actualDeserialized testMessages
	_ = json.Unmarshal(actualBody, &actualDeserialized)
	actual := actualDeserialized.Messages

	if len(expected) != len(actual) {
		return fmt.Errorf("different lengths: %d != %d\nexpected: %+q\nactual: %+q", len(expected), len(actual), expected, actual)
	}

	sort.Slice(expected, func(i, j int) bool {
		return expected[i] < expected[j]
	})
	sort.Slice(actual, func(i, j int) bool {
		return actual[i] < actual[j]
	})

	for i := range expected {
		if expected[i] != actual[i] {
			return fmt.Errorf("do not match in pos %d: %s != %s\nexpected: %+q\nactual: %+q", i, expected[i], actual[i], expected, actual)
		}
	}

	return nil
}
