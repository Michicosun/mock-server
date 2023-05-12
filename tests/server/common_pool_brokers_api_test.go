package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"mock-server/internal/server/protocol"
	"sort"
	"testing"
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
