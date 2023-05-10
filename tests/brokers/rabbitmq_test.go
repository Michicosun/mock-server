package brokers_test

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"reflect"
	"testing"
	"time"
)

func TestRabbitMq(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_brokers_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	handler, err := brokers.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool", "test-mock-queue"))
	if err != nil {
		t.Error(err)
	}

	readTaskId := handler.NewReadTask().Schedule()

	time.Sleep(1 * time.Second)

	handler, err = brokers.GetMessagePool("test-pool")
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("MessagePool: %s\n", handler)

	writeTaskId := handler.NewWriteTask([]string{"40", "41", "42"}).Schedule()

	time.Sleep(3 * time.Second)

	writeTaskMessages, err := database.GetTaskMessages(context.TODO(), string(writeTaskId))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(writeTaskMessages, []string{"40", "41", "42"}) {
		t.Errorf("res != expected: %+q != %+q", writeTaskMessages, []string{"40", "41", "42"})
	}

	readTaskMessages, err := database.GetTaskMessages(context.TODO(), string(readTaskId))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(readTaskMessages, []string{"40", "41", "42"}) {
		t.Errorf("res != expected: %+q != %+q", readTaskMessages, []string{"40", "41", "42"})
	}
}
