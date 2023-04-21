package broker_tests

// func TestScheduler(t *testing.T) {
// 	test_lib.InitTest()

// 	brokers.MPRegistry.Init()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	brokers.MPTaskScheduler.Init(ctx, configs.GetMPTaskSchedulerConfig())
// 	brokers.MPTaskScheduler.Start()

// 	_, err := brokers.MPRegistry.AddMessagePool(brokers.NewRabbitMQMessagePool("test-pool", "test-mock-queue"))
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if len(database.DB.GetReadMessagesCollection().GetAllKeys()) != 0 {
// 		t.Error(fmt.Errorf("incorrect setup"))
// 	}

// 	if len(database.DB.GetWriteMessagesCollection().GetAllKeys()) != 0 {
// 		t.Error(fmt.Errorf("incorrect setup"))
// 	}

// 	handler, err := brokers.MPRegistry.GetMessagePool("test-pool")
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	handler.NewWriteTask([][]byte{
// 		[]byte(fmt.Sprintf("%d", 40)),
// 		[]byte(fmt.Sprintf("%d", 41)),
// 		[]byte(fmt.Sprintf("%d", 42)),
// 	}).Schedule()

// 	<-time.After(1 * time.Second)

// 	handler.NewReadTask().Schedule()

// 	<-time.After(2 * time.Second)

// 	cancel()

// 	brokers.MPTaskScheduler.Stop()
// 	for x := range brokers.MPTaskScheduler.Errors() {
// 		t.Error(x)
// 	}

// 	if len(database.DB.GetReadMessagesCollection().GetAllKeys()) != 3 {
// 		t.Error(fmt.Errorf("read messages error"))
// 	}

// 	if len(database.DB.GetWriteMessagesCollection().GetAllKeys()) != 3 {
// 		t.Error(fmt.Errorf("write messages error"))
// 	}
// }
