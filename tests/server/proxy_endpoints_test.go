package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"testing"
)

func TestProxyRoutesSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	proxyApiEndpoint := endpoint + "/api/routes/proxy"

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
	code, body = DoGet(proxyApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// create route /test_url which proxies to /api/ping
	requestBody := []byte(`{
		"path": "/test_url",
		"proxy_url": "http://localhost:1337/api/ping"
	}`)
	code, _ = DoPost(proxyApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `http://localhost:1337/api/ping`
	code, body = DoGet(proxyApiEndpoint+"/proxy_url?path=/test_url", t)
	if code != 200 {
		t.Errorf("expected to be possible make request for proxy url")
	}

	if !bytes.Equal(body, []byte(`"http://localhost:1337/api/ping"`)) {
		t.Errorf(`dynamic data mismatch: %s != "http://localhost:1337/api/ping"`, body)
	}

	// expects `hello`
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if !bytes.Equal(body, []byte(`"Ping yourself, I have another work!"`)) {
		t.Errorf(`proxy data mismatch: %s != "hello"`, body)
	}

	// expects ["/test_url"]
	code, body = DoGet(proxyApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":["/test_url"]}`)) {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// update  created route /test_url, set proxying to /api/routes/proxy
	otherRequestBody := []byte(`{
		"path": "/test_url",
		"proxy_url": "http://localhost:1337/api/routes/proxy"
	}`)
	code = DoPut(proxyApiEndpoint, otherRequestBody, t)
	if code != 204 {
		t.Errorf("update route failed")
	}

	// expects ["/test_url"]
	code, body = DoGet(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to updated route")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":["/test_url"]}`)) {
		t.Errorf(`proxy data mismatch: %s != "hehe"`, body)
	}

	// detele /test_url
	code = DoDelete(proxyApiEndpoint+"?path=/test_url", t)
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
	code, body = DoGet(proxyApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if !bytes.Equal(body, []byte(`{"endpoints":[]}`)) {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}

func TestProxyRoutesDoublePost(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	proxyApiEndpoint := endpoint + "/api/routes/proxy"

	requestBody := []byte(`{
		"path": "/test_url",
		"proxy_url": "http://localhost:1337"
	}`)
	code, _ := DoPost(proxyApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}

	code, _ = DoPost(proxyApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}

	otherRequestBody := []byte(`{
		"path": "/test_url",
		"proxy_url": "http://localhost:1337"
	}`)
	code = DoPut(proxyApiEndpoint, otherRequestBody, t)
	if code != 204 {
		t.Errorf("expected to be possible to update already created endpoint: expected 204 != %d", code)
	}
}
func TestProxyRoutesBadRoute(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	proxyApiEndpoint := endpoint + "/api/routes/proxy"

	requestBodyBad := []byte(`{
		"path": "/test_url",
		"proxy_url": "localhost"
	}`)
	code, _ := DoPost(proxyApiEndpoint, requestBodyBad, t)
	if code != 400 {
		t.Errorf("expected to fail create because scheme is ommited: expected 400 != %d", code)
	}

	requestBody := []byte(`{
		"path": "/test_url",
		"proxy_url": "http://localhost:1337"
	}`)
	code, _ = DoPost(proxyApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}

	code = DoPut(proxyApiEndpoint, requestBodyBad, t)
	if code != 400 {
		t.Errorf("expected to fail update because scheme is ommited: expected 400 != %d", code)
	}

	code = DoPut(proxyApiEndpoint, requestBody, t)
	if code != 204 {
		t.Errorf("expected to be possible to update already created endpoint: expected 204 != %d", code)
	}
}
