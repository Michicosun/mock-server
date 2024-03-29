package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"testing"
)

func TestStaticRoutesSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	staticApiEndpoint := endpoint + "/api/routes/static"

	//////////////////////////////////////////////////////

	testUrl := endpoint + "/test_url"

	// no routes created -> 400
	code, body := DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected 400 on mismatch get")
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf(`mismatch get: %s != {"error":"no such path: /test_url"}`, body)
	}

	// expects []
	code, body = DoGet(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// create route /test_url with response `hello`
	requestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hello"
	}`)
	code, _ = DoPost(staticApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `hello`
	code, body = DoGet(staticApiEndpoint+"/expected_response?path=/test_url", t)
	if code != 200 {
		t.Errorf("expected to be possible make request for expected response")
	}

	if !bytes.Equal(body, []byte(`"hello"`)) {
		t.Errorf(`static data mismatch: %s != "hello"`, body)
	}

	// expects `hello`
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if !bytes.Equal(body, []byte(`"hello"`)) {
		t.Errorf(`static data mismatch: %s != "hello"`, body)
	}

	// expects ["/test_url"]
	code, body = DoGet(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":["/test_url"]}`)) {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// update  created route /test_url, set response `hehe`
	otherRequestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hehe"
	}`)
	code = DoPut(staticApiEndpoint, otherRequestBody, t)
	if code != 204 {
		t.Errorf("update route failed")
	}

	// expects `hehe`
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to updated route")
	}

	if !bytes.Equal(body, []byte(`"hehe"`)) {
		t.Errorf(`static data mismatch: %s != "hehe"`, body)
	}

	// detele /test_url
	code = DoDelete(staticApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("it must be possible to delete route")
	}

	// /test_url deleted -> 400
	code, body = DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected to be impossible to request deleted route: %d != 400, body = %s", code, body)
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf("unexpected error description")
	}

	// expects []
	code, body = DoGet(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}

func TestStaticRoutesDoublePost(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	staticApiEndpoint := endpoint + "/api/routes/static"

	requestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hello"
	}`)
	code, _ := DoPost(staticApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}

	code, _ = DoPost(staticApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}

	otherRequestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hello"
	}`)
	code = DoPut(staticApiEndpoint, otherRequestBody, t)
	if code != 204 {
		t.Errorf("expected to be possible to update already created endpoint: expected 204 != %d", code)
	}
}
