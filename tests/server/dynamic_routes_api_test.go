package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
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
	code, body := DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected 400 on mismatch get")
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf(`mismatch get: %s != {"error":"no such path: /test_url"}`, body)
	}

	// expects []
	code, body = DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// try to update non route that is not exists yet
	updateBody := []byte(`{
		"path": "/test_url",
		"code": "def func(headers, body):\n    return['noooo way']"
	}`)
	code = DoPut(dynamicApiEndpoint, updateBody, t)
	if code != 404 {
		t.Errorf(`expected 404 code on non created path`)
	}

	// create route /test_url with response `print(['noooo way', 123])`
	requestBody := []byte(`{
		"path": "/test_url",
		"code": "def func(headers, body):\n    return ['noooo way', 123]"
	}`)
	code, _ = DoPost(dynamicApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `def func():\n    print(['noooo way', 123])`
	code, body = DoGet(dynamicApiEndpoint+"/code?path=/test_url", t)
	if code != 200 {
		t.Errorf("expected to be possible make request for code")
	}

	if !bytes.Equal(body, []byte(`"def func(headers, body):\n    return ['noooo way', 123]\n"`)) {
		t.Errorf(`dynamic data mismatch: %s != "def func(headers, body):\n    return ["noooo way", 123]\n"`, body)
	}

	// expects `[\"noooo way\", 123]\n`
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if !bytes.Equal(body, []byte(`"[\"noooo way\", 123]"`)) {
		t.Errorf(`dynamic data mismatch: %s != ["noooo way", 123]`, body)
	}

	// update code
	code = DoPut(dynamicApiEndpoint, updateBody, t)
	if code != 204 {
		t.Errorf("update route's code failed")
	}

	// expects `['noooo way]`
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to an updated route")
	}

	if !bytes.Equal(body, []byte(`"[\"noooo way\"]"`)) {
		t.Errorf(`dynamic data mismatch: %s != [\"noooo way\"]`, body)
	}

	// expects ["/test_url"]
	code, body = DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":["/test_url"]}`)) {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// detele /test_url
	code = DoDelete(dynamicApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("it must be possible to delete route")
	}

	// /test_url deleted -> 400
	code, body = DoGet(testUrl, t)
	if code != 400 {
		t.Errorf("expected to be impossible to request deleted route: %d != 400", code)
	}

	if !bytes.Equal(body, []byte(`{"error":"no such path: /test_url"}`)) {
		t.Errorf("unexpected error description")
	}

	// expects []
	code, body = DoGet(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}

var TEST_DYN_HANDLE_CODE = []byte(`"def func(headers, body):\n    hA, hB = headers['A'], headers['B']\n    A, B, C = body['A'], body['B'], body['C']\n    return (hA[1], hB, A, int(B) - 3, list(reversed(C)))\n"`)

var TEST_QUERY_DYN_HANDLE = []byte(`
{
	"path": "/test_url",
	"code": "def func(headers, body):\n    hA, hB = headers['A'], headers['B']\n    A, B, C = body['A'], body['B'], body['C']\n    return (hA[1], hB, A, int(B) - 3, list(reversed(C)))"
}
`)

func TestDynamicRoutesScriptWithArgs(t *testing.T) {
	testScriptHeaders := map[string][]string{
		"A": {"A", "B"},
		"B": {"C"},
	}
	testScriptBody := []byte(`{
		"A": "hello, it's me",
		"B": 42,
		"C": ["a", "b", "c"]
	}`)
	expectedResponse := []byte(`("B", ["C"], "sample_A", 39, ["c", "b", "a"])`)

	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"
	testUrl := endpoint + "/test_url"

	code, _ := DoPost(dynamicApiEndpoint, TEST_QUERY_DYN_HANDLE, t)
	if code != 200 {
		t.Errorf("failed to add new dynamic route")
	}

	// expects TEST_DYN_HANDLE_CODE
	code, body := DoGet(dynamicApiEndpoint+"/code?path=/test_url", t)
	if code != 200 {
		t.Errorf("expected to be possible make request for code")
	}

	if !bytes.Equal(body, TEST_DYN_HANDLE_CODE) {
		t.Errorf("dynamic data mismatch:\n%s\n!=\n%s", body, TEST_DYN_HANDLE_CODE)
	}

	code, body = DoPostWithHeaders(testUrl, testScriptHeaders, testScriptBody, t)
	if code != 200 {
		t.Errorf("failed to query created dynamic route")
	}
	if bytes.Equal(body, expectedResponse) {
		t.Errorf(`dynamic data mismatch: %s != %s`, body, expectedResponse)
	}
}

func TestDynamiRoutesWithEmptyArgs(t *testing.T) {
	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(headers, body):\n    return [1, 2, 3]"
	}`)
	testScriptArgs := []byte(`{}`)
	expectedResponse := []byte(`[1, 2, 3]`)

	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"
	testUrl := endpoint + "/test_url"

	code, _ := DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("failed to add new dynamic route")
	}

	code, body := DoPost(testUrl, testScriptArgs, t)
	if code != 200 {
		t.Errorf("failed to query created dynamic route")
	}
	if bytes.Equal(body, expectedResponse) {
		t.Errorf(`dynamic data mismatch: %s != %s`, body, expectedResponse)
	}
}

func TestDynamicRoutesDoublePost(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"

	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(A, B, C):\n    pass"
	}`)
	code, _ := DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("create route failed: Expected 200 != %d", code)
	}

	code, _ = DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}

	otherTestBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(A, B, C):\n    pass"
	}`)
	code = DoPut(dynamicApiEndpoint, otherTestBodyScript, t)
	if code != 204 {
		t.Errorf("expected to be possible to update already created endpoint: expected 204 != %d", code)
	}
}

func TestDynamiRoutesBadScript(t *testing.T) {
	testBodyScriptBad := []byte(`{
		"path": "/test_url",
		"code": "def func(headers, body):print(A\n"
	}`)
	testScriptArgs := []byte(`{
		"A": 1
	}`)

	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"
	testUrl := endpoint + "/test_url"

	code, _ := DoPost(dynamicApiEndpoint, testBodyScriptBad, t)
	if code != 200 {
		t.Errorf("failed to add new dynamic route")
	}

	code, body := DoPost(testUrl, testScriptArgs, t)
	t.Logf("Received body: %s", string(body))
	if code != 400 {
		t.Errorf("expected to failed: 400 != %d", code)
	}
}

func TestDynamicRoutesBadArgs(t *testing.T) {
	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(headers, body):\n    print(body['A'], body['B'], body['C'])\n"
	}`)
	testScriptArgsBad := []byte(`{
		"A": 1,
		"B": 2
	}`)

	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"
	testUrl := endpoint + "/test_url"

	code, _ := DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("failed to add new dynamic route")
	}

	code, body := DoPost(testUrl, testScriptArgsBad, t)
	t.Logf("Received body: %s", string(body))
	if code != 400 {
		t.Errorf("expected to failed: 400 != %d", code)
	}
}
