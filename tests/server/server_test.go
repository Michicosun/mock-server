package server_test

import (
	"bytes"
	"fmt"
	"io"
	"mock-server/internal/configs"
	"mock-server/internal/control"
	"net/http"
	"testing"
)

func do_get(url string, t *testing.T) (int, string) {
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
		return 0, ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return 0, ""
	}

	return resp.StatusCode, string(body)
}

func do_post(url string, content []byte, t *testing.T) int {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(content))
	if err != nil {
		t.Error(err)
		return 0
	}

	return resp.StatusCode
}

func do_delete(url string, t *testing.T) int {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Error(err)
		return 0
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

func TestStaticRoutes(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	staticApiEndpoint := endpoint + "/api/routes/static"

	url := endpoint + "/api/ping"
	code, body := do_get(url, t)

	if code != 200 {
		t.Errorf("ping status code != 200")
	}

	if body != `"Ping yourself, I have another work!"` {
		t.Errorf(`ping: %s != "Ping yourself, I have another work!\n"`, body)
	}

	//////////////////////////////////////////////////////

	testUrl := endpoint + "/test_url"

	// no routes created -> 400
	code, body = do_get(testUrl, t)
	if code != 400 {
		t.Errorf("expected 400 on mismatch get")
	}

	if body != `{"error":"no such path: /test_url"}` {
		t.Errorf(`mismatch get: %s != {"error":"no such path: /test_url"}`, body)
	}

	// expects []
	code, body = do_get(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":[]}` {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// create route /test_url with reponse `hello`
	requestBody := []byte(`{
		"path": "/test_url",
		"response": "hello"
	}`)
	code = do_post(staticApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `hello`
	code, body = do_get(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if body != `"hello"` {
		t.Errorf(`static data mismatch: %s != "hello"`, body)
	}

	// expects ["/test_url"]
	code, body = do_get(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":["/test_url"]}` {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// detele /test_url
	code = do_delete(staticApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("it must be possible to delete route")
	}

	// /test_url deleted -> 404
	code, body = do_get(testUrl, t)
	if code != 400 {
		t.Errorf("expected to be impossible to request deleted route: %d != 400", code)
	}

	if body != `{"error":"no such path: /test_url"}` {
		t.Errorf("unexpected error description")
	}

	// expects []
	code, body = do_get(staticApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":[]}` {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}

func TestDynamicRoutes(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_server_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	cfg := configs.GetServerConfig()
	endpoint := fmt.Sprintf("http://%s", cfg.Addr)
	dynamicApiEndpoint := endpoint + "/api/routes/dynamic"

	url := endpoint + "/api/ping"
	code, body := do_get(url, t)

	if code != 200 {
		t.Errorf("ping status code != 200")
	}

	if body != `"Ping yourself, I have another work!"` {
		t.Errorf(`ping: %s != "Ping yourself, I have another work!\n"`, body)
	}

	//////////////////////////////////////////////////////

	testUrl := endpoint + "/test_url"

	// no routes created -> 400
	code, body = do_get(testUrl, t)
	if code != 400 {
		t.Errorf("expected 400 on mismatch get")
	}

	if body != `{"error":"no such path: /test_url"}` {
		t.Errorf(`mismatch get: %s != {"error":"no such path: /test_url"}`, body)
	}

	// expects []
	code, body = do_get(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":[]}` {
		t.Errorf(`list request must be empty at the begining: %s != {"endpoints":[]}`, body)
	}

	// create route /test_url with reponse `print(['noooo way', 123])`
	requestBody := []byte(`{
		"path": "/test_url",
		"code": "print(['noooo way', 123])"
	}`)
	code = do_post(dynamicApiEndpoint, requestBody, t)
	if code != 200 {
		t.Errorf("create route failed")
	}

	// expects `hello`
	code, body = do_get(testUrl, t)
	if code != 200 {
		t.Errorf("expected to be possible make request to new route")
	}

	if body != `['noooo way', 123]` {
		t.Errorf(`dynamic data mismatch: %s != ['noooo way', 123]`, body)
	}

	// expects ["/test_url"]
	code, body = do_get(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":["/test_url"]}` {
		t.Errorf(`must be visible new route after creation: %s != {"endpoints":["/test_url"]}`, body)
	}

	// detele /test_url
	code = do_delete(dynamicApiEndpoint+"?path=/test_url", t)
	if code != 204 {
		t.Errorf("it must be possible to delete route")
	}

	// /test_url deleted -> 404
	code, body = do_get(testUrl, t)
	if code != 400 {
		t.Errorf("expected to be impossible to request deleted route: %d != 400", code)
	}

	if body != `{"error":"no such path: /test_url"}` {
		t.Errorf("unexpected error description")
	}

	// expects []
	code, body = do_get(dynamicApiEndpoint, t)
	if code != 200 {
		t.Errorf("expected 200 code response on list all request")
	}

	if body != `{"endpoints":[]}` {
		t.Errorf(`expected empty response after deletion: %s != {"endpoints":[]}`, body)
	}
}
