package brokers_test

import (
	"context"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"mock-server/internal/util"
	"testing"
	"time"
)

var TEST_SCRIPT = util.WrapCodeForEsb(`
def func(msgs):
	print(msgs[::-1])
`)
var TEST_ARGS = []string{"msg1", "msg2", "msg3"}

func TestEsb(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_brokers_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	pool1, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-1", "test-mock-queue-1"))
	if err != nil {
		t.Error(err)
	}

	pool2, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-2", "test-mock-queue-2"))
	if err != nil {
		t.Error(err)
	}

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	err = fs.Write("mapper", "test-mapper.py", TEST_SCRIPT)
	if err != nil {
		t.Error(err)
	}

	err = brokers.Esb.AddEsbRecordWithMapper("test-pool-1", "test-pool-2", "test-mapper.py")
	if err != nil {
		t.Error(err)
	}

	///////////////////////////////////////////////////////////////////////////////////////

	// schedule read -- wait for args
	readTaskId1 := pool1.NewReadTask().Schedule()

	time.Sleep(2 * time.Second)

	// push args to first pool
	writeTaskId := pool1.NewWriteTask(TEST_ARGS).Schedule()

	time.Sleep(2 * time.Second)

	// schedule read -- pull script invocation result
	readTaskId2 := pool2.NewReadTask().Schedule()

	time.Sleep(20 * time.Second)

	res, err := database.GetTaskMessages(context.TODO(), string(readTaskId1))
	if err != nil {
		t.Error(err)
	}
	t.Errorf("%s = %s\n", readTaskId1, res)
	res, err = database.GetTaskMessages(context.TODO(), string(writeTaskId))
	if err != nil {
		t.Error(err)
	}
	t.Errorf("%s = %s\n", writeTaskId, res)
	res, err = database.GetTaskMessages(context.TODO(), string(readTaskId2))
	if err != nil {
		t.Error(err)
	}
	t.Errorf("%s = %s\n", readTaskId2, res)
}
