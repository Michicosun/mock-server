package brokers_test

import (
	"context"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"mock-server/internal/util"
	"reflect"
	"testing"
	"time"
)

var TEST_SCRIPT = util.WrapCodeForEsb(`
def func(msgs):
	return msgs[::-1]
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

	pool1, err := brokers.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-1", "test-mock-queue-1"))
	if err != nil {
		t.Error(err)
	}

	pool2, err := brokers.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool-2", "test-mock-queue-2"))
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

	err = brokers.AddEsbRecordWithMapper(context.TODO(), "test-pool-1", "test-pool-2", "test-mapper.py")
	if err != nil {
		t.Error(err)
	}

	///////////////////////////////////////////////////////////////////////////////////////

	// schedule read -- wait for args
	readTaskId1 := pool1.NewReadTask().Schedule()

	time.Sleep(3 * time.Second)

	// push args to first pool
	writeTaskId := pool1.NewWriteTask(TEST_ARGS).Schedule()

	time.Sleep(5 * time.Second)

	// schedule read -- pull script invocation result
	readTaskId2 := pool2.NewReadTask().Schedule()

	time.Sleep(5 * time.Second)

	res, err := database.GetTaskMessages(context.TODO(), string(readTaskId1))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(res, []string{"msg1", "msg2", "msg3"}) {
		t.Errorf("res != expected: %+q != %+q", res, []string{"msg1", "msg2", "msg3"})
	}
	res, err = database.GetTaskMessages(context.TODO(), string(writeTaskId))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(res, []string{"msg1", "msg2", "msg3"}) {
		t.Errorf("res != expected: %+q != %+q", res, []string{"msg1", "msg2", "msg3"})
	}
	res, err = database.GetTaskMessages(context.TODO(), string(readTaskId2))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(res, []string{"msg3", "msg2", "msg1"}) {
		t.Errorf("res != expected: %+q != %+q", res, []string{"msg3", "msg2", "msg1"})
	}
}
