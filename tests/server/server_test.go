package server_test

import (
	"bytes"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	hlp "mock-server/internal/test_helpers"
	"testing"
)

func TestServerSimple(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)

	url := endpoint + "/api/ping"
	code, body := hlp.DoGet(url, t)

	if code != 200 {
		t.Errorf("ping status code != 200")
	}

	if !bytes.Equal(body, []byte(`"Ping yourself, I have another work!"`)) {
		t.Errorf(`ping: %s != "Ping yourself, I have another work!\n"`, body)
	}
}

func TestServerSameNamespaceForEndpoints(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	staticApiEndpoint := endpoint + "/api/routes/static"
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"

	requestBody := []byte(`{
		"path": "/test_url",
		"expected_response": "hello"
	}`)
	testBodyScript := []byte(`{
		"path": "/test_url",
		"code": "def func(A, B, C):\n    pass"
	}`)

	// post static endpoint than try to post same static and dynamic
	code := hlp.DoPost(staticApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}
	code = hlp.DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
	code = hlp.DoPost(staticApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}

	// wipe
	code = hlp.DoDelete(staticApiEndpoint, t)
	if code != 204 {
		t.Errorf("expected to be possible to delete existing endpoint")
	}

	// same as first test but fisrt post dynamic
	code = hlp.DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 200 {
		t.Errorf("create route failed: expected 200 != %d", code)
	}
	code = hlp.DoPost(staticApiEndpoint, requestBody, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
	code = hlp.DoPost(dynamicApiEndpoint, testBodyScript, t)
	if code != 409 {
		t.Errorf("expected to receive conflict: expected 409 != %d", code)
	}
}
