package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"testing"
)

func TestEsbBrokersSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	esbApiEndpoint := endpoint + "/api/brokers/esb"

	//////////////////////////////////////////////////////

	// expects []
	code, body := DoGet(esbApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all esb records")
	}

	if !bytes.Equal(body, []byte(`{"records":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"pools":[]}`, body)
	}

	// create esb record without mapper script
	esbRecord := []byte(`{"pool_name_in":"pool_in","pool_name_out":"pool_out"}`)

	code, body = DoPost(esbApiEndpoint, esbRecord, t)
	if code != 200 {
		t.Errorf("create esb record failed: %s", body)
	}

	// expects get that esb record
	code, body = DoGet(esbApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 response on list all esb records: %s", body)
	}

	expectedRecordsList := fmt.Sprintf(`{"records":[%s]}`, esbRecord)
	if !bytes.Equal(body, []byte(expectedRecordsList)) {
		t.Errorf(`list request must contain created esb record: %s != %s`, body, expectedRecordsList)
	}

	// create esb record with mapper
	esbRecordWithMapper := []byte(`{
		"pool_name_in":"pool_in_mapper",
		"pool_name_out":"pool_out_mapper",
		"code":"def func(msgs):\n    return msgs[::-1]"
	}`)

	code, body = DoPost(esbApiEndpoint, esbRecordWithMapper, t)
	if code != 200 {
		t.Errorf("create esb record failed: %s", body)
	}

	// get mapper code
	code, body = DoGet(esbApiEndpoint+"/code?pool_in=pool_in_mapper", t)
	if code != 200 {
		t.Errorf("failed to get mapper code: %s", body)
	}

	if !bytes.Equal(body, []byte(`"def func(msgs):\n    return msgs[::-1]\n"`)) {
		t.Errorf(`expected to get created code: %s != "def func(msgs):\n    return msgs[::-1]\n"`, body)
	}

	// expects get both esb records
	code, body = DoGet(esbApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 response on list all esb records: %s", body)
	}

	expectedRecordsList = fmt.Sprintf(`{"records":[%s,%s]}`, esbRecord, []byte(`{"pool_name_in":"pool_in_mapper","pool_name_out":"pool_out_mapper"}`))
	if !bytes.Equal(body, []byte(expectedRecordsList)) {
		t.Errorf(`list request must contain created esb record: %s != %s`, body, expectedRecordsList)
	}

	// delete second esb record
	code = DoDelete(esbApiEndpoint+"?pool_in=pool_in_mapper", t)
	if code != 204 {
		t.Errorf("delete esb record failed")
	}

	// expects get only first esb record
	code, body = DoGet(esbApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 response on list all esb records: %s", body)
	}

	expectedRecordsList = fmt.Sprintf(`{"records":[%s]}`, esbRecord)
	if !bytes.Equal(body, []byte(expectedRecordsList)) {
		t.Errorf(`list request must contain only first esb record: %s != %s`, body, expectedRecordsList)
	}
}
func TestEsbBrokersDoublePost(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	esbApiEndpoint := endpoint + "/api/brokers/esb"

	//////////////////////////////////////////////////////

	esbRecord := []byte(`{
		"pool_name_in": "pool_in",
		"pool_name_out": "pool_out"
	}`)
	esbRecordWithMapper := []byte(`{
		"pool_name_in": "pool_in",
		"pool_name_out": "pool_out_mapper",
		"code": "def func(msgs):\n    return msgs[::-1]"
	}`)

	code, body := DoPost(esbApiEndpoint, esbRecord, t)
	if code != 200 {
		t.Errorf("create esb record failed: %s", body)
	}

	code, body = DoPost(esbApiEndpoint, esbRecord, t)
	if code != 409 {
		t.Errorf("expected to be impossible to create esb record with the same in-pool name: %s", body)
	}

	code, body = DoPost(esbApiEndpoint, esbRecordWithMapper, t)
	if code != 409 {
		t.Errorf("expected to be impossible to create esb record with the same in-pool name even if it has a mapper: %s", body)
	}
}
