package broker_tests

import (
	"fmt"
	"mock-server/internal/brokers"
	"mock-server/internal/control"
	"mock-server/internal/util"
	"testing"
	"time"
)

func TestEsb(t *testing.T) {
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

	err = fs.Write("mapper", "test-mapper.py", []byte(`print([[72, 69, 76, 76, 79], [87, 79, 82, 76, 68], [33]])`))
	if err != nil {
		t.Error(err)
	}

	err = brokers.Esb.AddEsbRecordWithMapper("test-pool-1", "test-pool-2", "test-mapper.py")
	if err != nil {
		t.Error(err)
	}

	///////////////////////////////////////////////////////////////////////////////////////

	pool1.NewReadTask().Schedule()

	time.Sleep(2 * time.Second)

	pool1.NewWriteTask([][]byte{
		[]byte(fmt.Sprintf("%d", 40)),
		[]byte(fmt.Sprintf("%d", 41)),
		[]byte(fmt.Sprintf("%d", 42)),
	}).Schedule()

	time.Sleep(2 * time.Second)

	pool2.NewReadTask().Schedule()

	time.Sleep(10 * time.Second)
}
