package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	hlp "mock-server/internal/test_helpers"
	"testing"
)

func TestDynamicRoutesSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"

	//////////////////////////////////////////////////////

	testUrl := endpoint + "/test_url"

	// no routes created -> 400
	code, body := hlp.DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected 400 on mismatch get")
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf(`mismatch get: %s != {"error":"no such path: /test_url"}`, body)
	}

	// expects []
	code, body = hlp.DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// try to update non route that is not exists yet
	updateBody := []byte(`{
		"path": "/test_url",
		"code": "def func():\n    print(['noooo way'])"
	}`)
	code = hlp.DoPut(dynamicApiEndpoint, updateBody, t)
	if code != 404 {
		t.Errorf(`expected 404 code on non created path`)
	}

	// create route /test_url with response `print(['noooo way', 123])`
	requestBody := []byte(`{
		"path": "/test_url",
		"code": "def func():\n    print(['noooo way', 123])"
	}`)
	code = hlp.DoPost(dynamicApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `[\"noooo way\", 123]\n`
	code, body = hlp.DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if !bytes.Equal(body, []byte(`"[\"noooo way\", 123]"`)) {
		t.Errorf(`dynamic data mismatch: %s != ["noooo way", 123]`, body)
	}

	// update code
	code = hlp.DoPut(dynamicApiEndpoint, updateBody, t)
	if code != 204 {
		t.Errorf("update route's code failed")
	}

	// expects `['noooo way]`
	code, body = hlp.DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to an updated route")
	}

	if !bytes.Equal(body, []byte(`"[\"noooo way\"]"`)) {
		t.Errorf(`dynamic data mismatch: %s != [\"noooo way\"]`, body)
	}

	// expects ["/test_url"]
	code, body = hlp.DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":["/test_url"]}`)) {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// detele /test_url
	code = hlp.DoDelete(dynamicApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("it must be possible to delete route")
	}

	// /test_url deleted -> 404
	code, body = hlp.DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected to be impossible to request deleted route: %d != 400", code)
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf("unexpected error description")
	}

	// expects []
	code, body = hlp.DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}

// TODO: Add test that examines run request on bad python code (e.g. with syntax errors)
// N.B. Currently, such request will cause an interanl error (500), but client error is needed

func TestDynamicRoutesScriptWithArgs(t *testing.T) {
	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(A, B, C):\n    print(A)\n    print(B - 3)\n    print(list(reversed(C)))\n"}`)
	testScriptArgs := []byte(`{
		"A": "hello, it's me",
		"B": 42,
		"C": ["a", "b", "c"]
	}`)
	expectedResponse := []byte("hello, it's me\n39\n[\"c\", \"b\", \"a\"]")

	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"
	testUrl := endpoint + "/test_url"

	code := hlp.DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("failed to add new dynamic route")
	}

	code, body := hlp.DoGetWithBody(testUrl, testScriptArgs, t)
	if code != 200 {
		t.Errorf("failed to query created dynamic route")
	}
	if bytes.Equal(body, expectedResponse) {
		t.Errorf(`dynamic data mismatch: %s != %s`, body, expectedResponse)
	}
}
