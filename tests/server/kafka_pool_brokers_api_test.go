package server_test

// func TestPoolBrokersKafkaTaskSchedulingSimple(t *testing.T) {
// 	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

// 	control.Components.Start()
// 	defer control.Components.Stop()

// 	go func() {
// 		for err := range brokers.MPTaskScheduler.Errors() {
// 			t.Error(err)
// 		}
// 	}()

// 	cfg := configs.GetServerConfig()
// 	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
// 	poolApiEndpoint := endpoint + "/api/brokers/pool"

// 	//////////////////////////////////////////////////////

// 	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

// 	code, _ := DoPost(poolApiEndpoint, kafkaPool, t)
// 	if code != 200 {
// 		t.Errorf("create pool failed")
// 	}

// 	// schedule write than schedule read
// 	messages := []string{"msg1", "msg2", "msg3"}
// 	writeTask := createWriteTaskBody("pool", messages)
// 	code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
// 	if code != 204 {
// 		t.Errorf("schedule write task failed: %s", body)
// 	}
// 	time.Sleep(2 * time.Second)
// 	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
// 	if code != 204 {
// 		t.Errorf("schedule read task failed: %s", body)
// 	}

// 	time.Sleep(10 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query write tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected to write messages be available almost simultaneously after write task request: %s", err.Error())
// 	}

// 	time.Sleep(10 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query read tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected completed read task after some time: %s", err.Error())
// 	}

// 	// schedule read than schedule write
// 	moreMessages := []string{"msg4", "msg5"}
// 	messages = append(messages, moreMessages...)
// 	writeTask = createWriteTaskBody("pool", moreMessages)
// 	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
// 	if code != 204 {
// 		t.Errorf("schedule read task failed: %s", body)
// 	}
// 	time.Sleep(2 * time.Second)
// 	code, body = DoPost(poolApiEndpoint+"/write", writeTask, t)
// 	if code != 204 {
// 		t.Errorf("schedule write task failed: %s", body)
// 	}

// 	time.Sleep(10 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query write tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected to write messages be available almost simultaneously after write task request: %s", err.Error())
// 	}

// 	time.Sleep(1 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query read tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected completed read task after some time: %s", err.Error())
// 	}
// }

// func TestPoolBrokersKafkaManyWrites(t *testing.T) {
// 	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

// 	control.Components.Start()
// 	defer control.Components.Stop()

// 	go func() {
// 		for err := range brokers.MPTaskScheduler.Errors() {
// 			t.Error(err)
// 		}
// 	}()

// 	cfg := configs.GetServerConfig()
// 	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
// 	poolApiEndpoint := endpoint + "/api/brokers/pool"

// 	//////////////////////////////////////////////////////

// 	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

// 	code, body := DoPost(poolApiEndpoint, kafkaPool, t)
// 	if code != 200 {
// 		t.Errorf("create pool failed: %s", body)
// 	}

// 	const MESSAGE_COUNT = 10
// 	messages := make([]string, 0)
// 	for i := 0; i < MESSAGE_COUNT; i++ {
// 		messages = append(messages, fmt.Sprintf("msg%d", i))
// 	}

// 	// populate MESSAGE_COUNT / 2 write tasks
// 	for i := 0; i < MESSAGE_COUNT; i += 2 {
// 		writeTask := createWriteTaskBody("pool", messages[i:i+2])
// 		code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
// 		if code != 204 {
// 			t.Errorf("schedule write task failed: %s", body)
// 		}
// 	}
// 	time.Sleep(2 * time.Second)
// 	// schedule read task
// 	code, body = DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
// 	if code != 204 {
// 		t.Errorf("schedule write task failed: %s", body)
// 	}

// 	time.Sleep(15 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query write tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
// 	}

// 	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query read tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected completed read task after some time: %s", err.Error())
// 	}
// }

// func TestPoolBrokersKafkaFloodReads(t *testing.T) {
// 	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

// 	control.Components.Start()
// 	defer control.Components.Stop()

// 	go func() {
// 		for err := range brokers.MPTaskScheduler.Errors() {
// 			t.Error(err)
// 		}
// 	}()

// 	cfg := configs.GetServerConfig()
// 	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
// 	poolApiEndpoint := endpoint + "/api/brokers/pool"

// 	//////////////////////////////////////////////////////

// 	kafkaPool := []byte(`{"pool_name":"pool","topic_name":"queue","broker":"kafka"}`)

// 	code, body := DoPost(poolApiEndpoint, kafkaPool, t)
// 	if code != 200 {
// 		t.Errorf("create pool failed: %s", body)
// 	}

// 	const MESSAGE_COUNT = 20
// 	messages := make([]string, 0)
// 	for i := 0; i < MESSAGE_COUNT; i++ {
// 		messages = append(messages, fmt.Sprintf("msg%d", i))
// 	}

// 	// populate MESSAGE_COUNT / 4 read tasks
// 	for i := 0; i < MESSAGE_COUNT/4; i++ {
// 		code, body := DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
// 		if code != 204 {
// 			t.Errorf("schedule read task failed: %s", body)
// 		}
// 	}

// 	time.Sleep(2 * time.Second)

// 	// populate MESSAGE_COUNT write tasks
// 	for i := 0; i < MESSAGE_COUNT; i++ {
// 		writeTask := createWriteTaskBody("pool", messages[i:i+1])
// 		code, body := DoPost(poolApiEndpoint+"/write", writeTask, t)
// 		if code != 204 {
// 			t.Errorf("schedule write task failed: %s", body)
// 		}
// 	}

// 	time.Sleep(2 * time.Second)

// 	// populate MESSAGE_COUNT / 4 read tasks
// 	for i := 0; i < MESSAGE_COUNT/4; i++ {
// 		code, body := DoPost(poolApiEndpoint+"/read?pool=pool", []byte{}, t)
// 		if code != 204 {
// 			t.Errorf("schedule read task failed: %s", body)
// 		}
// 	}

// 	time.Sleep(10 * time.Second)

// 	code, body = DoGet(poolApiEndpoint+"/write?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query write tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
// 	}

// 	code, body = DoGet(poolApiEndpoint+"/read?pool=pool", t)
// 	if code != 200 {
// 		t.Errorf("Failed to query read tasks: %s", body)
// 	}
// 	if err := compareRequestMessagesResponse(messages, body); err != nil {
// 		t.Errorf("Expected completed read task after some time: %s", err.Error())
// 	}
// }

// func TestPoolBrokersKafkaManyPools(t *testing.T) {
// 	t.Setenv("CONFIG_PATH", "/configs/test_server_pool_api_config.yaml")

// 	control.Components.Start()
// 	defer control.Components.Stop()

// 	go func() {
// 		for err := range brokers.MPTaskScheduler.Errors() {
// 			t.Error(err)
// 		}
// 	}()

// 	cfg := configs.GetServerConfig()
// 	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
// 	poolApiEndpoint := endpoint + "/api/brokers/pool"

// 	//////////////////////////////////////////////////////

// 	const POOL_COUNT = 2
// 	const MESSAGE_COUNT_PER_POOL = 10
// 	var wg sync.WaitGroup
// 	wg.Add(POOL_COUNT)
// 	for poolNum := 0; poolNum < POOL_COUNT; poolNum++ {
// 		poolName := strconv.Itoa(poolNum)
// 		kafkaPool := []byte(fmt.Sprintf(`{
// 			"pool_name": "pool%s",
// 			"topic_name": "queue%s",
// 			"broker":"kafka"
// 		}`, poolName, poolName))

// 		go func() {
// 			code, body := DoPost(poolApiEndpoint, kafkaPool, t)
// 			if code != 200 {
// 				t.Errorf("create pool failed: %s", body)
// 			}

// 			messages := make([]string, 0)
// 			for i := 0; i < MESSAGE_COUNT_PER_POOL; i++ {
// 				messages = append(messages, fmt.Sprintf("msg%d", i))
// 			}

// 			// schedule write task
// 			writeTask := createWriteTaskBody("pool"+poolName, messages)
// 			code, body = DoPost(poolApiEndpoint+"/write", writeTask, t)
// 			if code != 204 {
// 				t.Errorf("schedule write task failed: %s", body)
// 			}

// 			time.Sleep(2 * time.Second)

// 			// schedule read task
// 			code, body = DoPost(poolApiEndpoint+"/read?pool=pool"+poolName, []byte{}, t)
// 			if code != 204 {
// 				t.Errorf("schedule read task failed: %s", body)
// 			}

// 			time.Sleep(15 * time.Second)

// 			code, body = DoGet(poolApiEndpoint+"/write?pool=pool"+poolName, t)
// 			if code != 200 {
// 				t.Errorf("Failed to query write tasks: %s", body)
// 			}
// 			if err := compareRequestMessagesResponse(messages, body); err != nil {
// 				t.Errorf("Expected to write messages be available after write task request: %s", err.Error())
// 			}

// 			code, body = DoGet(poolApiEndpoint+"/read?pool=pool"+poolName, t)
// 			if code != 200 {
// 				t.Errorf("Failed to query read tasks: %s", body)
// 			}
// 			if err := compareRequestMessagesResponse(messages, body); err != nil {
// 				t.Errorf("Expected completed read task after some time: %s", err.Error())
// 			}

// 			wg.Done()
// 		}()
// 	}

// 	wg.Wait()
// }
