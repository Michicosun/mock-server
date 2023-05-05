package coderun_test

import (
	"bytes"
	"mock-server/internal/coderun"
	"mock-server/internal/control"
	"mock-server/internal/util"
	"testing"
)

var TEST_SCRIPT = util.DecorateCodeForArgExtraction(`
def func(A, B, C):
	print(A)
	print(B - 3)
	print(list(reversed(C)))
`)

const EXPECTED_OUTPUT = "sample_A\n39\n['c', 'b', 'a']\n"

func TestCoderun(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_coderun_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	err = fs.Write("mapper", "test.py", []byte(TEST_SCRIPT))
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 1; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("mapper", "test.py", []byte(`{
			"A": "sample_A",
			"B": 42,
			"C": ["a", "b", "c"]
		}`))
		if err != nil {
			t.Error(err)
			return
		}

		if !bytes.Equal(out, []byte(EXPECTED_OUTPUT)) {
			t.Errorf(`%s != %s`, string(out), EXPECTED_OUTPUT)
		}

		worker.Return()
	}
}
