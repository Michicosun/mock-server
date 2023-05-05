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
