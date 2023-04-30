package util_tests

import (
	"mock-server/internal/control"
	"mock-server/internal/util"
	"testing"
)

func TestStorage(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_util_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	err = fs.Write("mapper", "test.py", []byte(`print("Hello, world!")`))
	if err != nil {
		t.Error(err)
	}

	s, err := fs.Read("mapper", "test.py")
	if err != nil {
		t.Error(err)
	}

	if s != `print("Hello, world!")` {
		t.Errorf(`%s != print("Hello, world!")`, s)
	}
}
