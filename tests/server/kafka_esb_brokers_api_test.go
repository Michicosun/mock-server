package server_test

import (
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"testing"
	"time"
)

func TestEsbBrokersKafkaScheduling(t *testing.T) {
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
	esbApiEndpoint := endpoint + "/api/brokers/esb"
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	// create rabbitmq pool
	kafkaPoolIn := []byte(`{
		"pool_name": "kafka_pool_in",
		"topic_name": "kafka_queue_in_scheduling",
		"broker": "kafka"
	}`)
	code, body := DoPost(poolApiEndpoint, kafkaPoolIn, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	kafkaPoolOut := []byte(`{
		"pool_name": "kafka_pool_out",
		"topic_name": "kafka_queue_out_scheduling",
		"broker": "kafka"
	}`)
	code, body = DoPost(poolApiEndpoint, kafkaPoolOut, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	// create esb record without mapper script
	esbRecord := []byte(`{
		"pool_name_in": "kafka_pool_in",
		"pool_name_out": "kafka_pool_out"
	}`)

	code, body = DoPost(esbApiEndpoint, esbRecord, t)
	if code != 200 {
		t.Errorf("create esb record failed: %s", body)
	}

	// submit messages to first pool
	messages := []string{"msg1", "msg2", "msg3"}
	writeTask := createWriteTaskBody("kafka_pool_in", messages)
	code, body = DoPost(esbApiEndpoint+"/task", writeTask, t)
	if code != 204 {
		t.Errorf("task submit failed: %s", body)
	}

	time.Sleep(10 * time.Second)

	code, body = DoPost(poolApiEndpoint+"/read?pool=kafka_pool_out", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}

	time.Sleep(10 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/read?pool=kafka_pool_out", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(messages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}
}

func TestEsbBrokersKafkaSchedulingWithMapper(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_esb_api_config.yaml")

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
	esbApiEndpoint := endpoint + "/api/brokers/esb"
	poolApiEndpoint := endpoint + "/api/brokers/pool"

	//////////////////////////////////////////////////////

	// create kafka pool
	kafkaPoolIn := []byte(`{
		"pool_name": "kafka_pool_in",
		"topic_name": "kafka_queue_in_with_mapper",
		"broker": "kafka"
	}`)
	code, body := DoPost(poolApiEndpoint, kafkaPoolIn, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	kafkaPoolOut := []byte(`{
		"pool_name": "kafka_pool_out",
		"topic_name": "kafka_queue_out_with_mapper",
		"broker": "kafka"
	}`)
	code, body = DoPost(poolApiEndpoint, kafkaPoolOut, t)
	if code != 200 {
		t.Errorf("create pool failed: %s", body)
	}

	// create esb record without mapper script
	esbRecordWithMapper := []byte(`{
		"pool_name_in": "kafka_pool_in",
		"pool_name_out": "kafka_pool_out",
		"code": "def func(msgs):\n    return msgs[::-1]"
	}`)

	code, body = DoPost(esbApiEndpoint, esbRecordWithMapper, t)
	if code != 200 {
		t.Errorf("create esb record failed: %s", body)
	}

	// submit messages to first pool
	messages := []string{"msg1", "msg2", "msg3"}
	expectedMessages := []string{"msg3", "msg2", "msg1"}
	writeTask := createWriteTaskBody("kafka_pool_in", messages)
	code, body = DoPost(esbApiEndpoint+"/task", writeTask, t)
	if code != 204 {
		t.Errorf("task submit failed: %s", body)
	}

	time.Sleep(10 * time.Second)

	code, body = DoPost(poolApiEndpoint+"/read?pool=kafka_pool_out", []byte{}, t)
	if code != 204 {
		t.Errorf("schedule read task failed: %s", body)
	}

	time.Sleep(10 * time.Second)

	code, body = DoGet(poolApiEndpoint+"/read?pool=kafka_pool_out", t)
	if code != 200 {
		t.Errorf("Failed to query read tasks: %s", body)
	}
	if err := compareRequestMessagesResponse(expectedMessages, body); err != nil {
		t.Errorf("Expected completed read task after some time: %s", err.Error())
	}
}
