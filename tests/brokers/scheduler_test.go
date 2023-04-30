package broker_tests

import (
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_brokers_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	handler, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool", "test-mock-queue"))
	if err != nil {
		t.Error(err)
	}

	handler.NewReadTask().Schedule()

	time.Sleep(1 * time.Second)

	handler, err = brokers.MPRegistry.GetMessagePool("test-pool")
	if err != nil {
		t.Error(err)
	}

	handler.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	time.Sleep(1 * time.Second)

	// TODO check database for read records
}
