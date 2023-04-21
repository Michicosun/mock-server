package broker_tests

import (
	"context"
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/coderun"
	"mock-server/internal/configs"
	"mock-server/internal/util"
	test_lib "mock-server/tests/common"
	"testing"
	"time"
)

func TestEsb(t *testing.T) {
	test_lib.InitTest()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	brokers.MPRegistry.Init()
	brokers.Esb.Init()

	brokers.MPTaskScheduler.Init(ctx, configs.GetMPTaskSchedulerConfig())
	brokers.MPTaskScheduler.Start()
	defer brokers.MPTaskScheduler.Stop()

	err := coderun.WorkerWatcher.Init(ctx, configs.GetCoderunConfig())
	if err != nil {
		t.Error(err)
	}

	defer coderun.WorkerWatcher.Stop()

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

	fs.Write("mapper", "test-mapper.py", []byte(`print([[72, 69, 76, 76, 79], [87, 79, 82, 76, 68], [33]])`))

	brokers.Esb.AddEsbRecordWithMapper("test-pool-1", "test-pool-2", "test-mapper.py")

	///////////////////////////////////////////////////////////////////////////////////////

	go func() {
		for err := range brokers.MPTaskScheduler.Errors() {
			t.Error(err)
		}
	}()

	pool1.NewReadTask().Schedule()

	<-time.After(2 * time.Second)

	pool1.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	<-time.After(2 * time.Second)

	pool2.NewReadTask().Schedule()

	<-time.After(10 * time.Second)

	cancel()

}
