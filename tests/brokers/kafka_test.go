package brokers_test

import (
	"context"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"reflect"
	"testing"
	"time"
)

func TestKafka(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_brokers_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	handler, err := brokers.MPRegistry.AddMessagePool(brokers.NewKafkaMessagePool("test-pool", "test-topic"))
	if err != nil {
		t.Error(err)
	}

	handler.NewWriteTask([]string{"40", "41", "42"}).Schedule()

	time.Sleep(3 * time.Second)

	handler, err = brokers.MPRegistry.GetMessagePool("test-pool")
	if err != nil {
		t.Error(err)
	}

	handler.NewReadTask().Schedule()

	time.Sleep(15 * time.Second)

	writeTaskMessages, err := database.GetTaskMessages(context.TODO(), "kafka:test-pool:test-topic:write")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(writeTaskMessages, []string{"40", "41", "42"}) {
		t.Errorf("res != expected: %+q != %+q", writeTaskMessages, []string{"40", "41", "42"})
	}

	readTaskMessages, err := database.GetTaskMessages(context.TODO(), "kafka:test-pool:test-topic:read")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(readTaskMessages, []string{"40", "41", "42"}) {
		t.Errorf("res != expected: %+q != %+q", readTaskMessages, []string{"40", "41", "42"})
	}
}
