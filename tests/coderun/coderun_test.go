package coderun_test

import (
	"bytes"
	"mock-server/internal/coderun"
	"mock-server/internal/control"
	"mock-server/internal/util"
	"testing"
)

var TEST_SCRIPT_DYN_HANDLE = util.WrapCodeForDynHandle(`
def func(A, B, C):
	print(A)
	print(B - 3)
	print(list(reversed(C)))
`)
var TEST_ARGS_DYN_HANDLE = coderun.NewDynHandleArgs([]byte(`
{
	"A": "sample_A",
	"B": 42,
	"C": ["a", "b", "c"]
}
`))
var EXPECTED_OUTPUT_DYN_HANDLE = "sample_A\n39\n[\"c\", \"b\", \"a\"]"

func TestCoderunForDynHandle(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_coderun_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	if err = fs.Write("dyn_handle", "test_dyn_handle.py", TEST_SCRIPT_DYN_HANDLE); err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("dyn_handle", "test_dyn_handle.py", TEST_ARGS_DYN_HANDLE)
		if err != nil {
			t.Error(err)
			return
		}

		if !bytes.Equal(out, []byte(EXPECTED_OUTPUT_DYN_HANDLE)) {
			t.Errorf(`%s != %s`, string(out), EXPECTED_OUTPUT_DYN_HANDLE)
		}

		worker.Return()
	}
}

var TEST_SCRIPT_ESB = util.WrapCodeForEsb(`
def func(msgs):
	print(msgs[::-1])
`)
var TEST_ARGS_ESB = coderun.NewMapperArgs([]string{"msg1", "msg2", "msg3"})
var EXPECTED_OUTPUT_ESB = `["msg3", "msg2", "msg1"]`

func TestCoderunForEsb(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_coderun_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	if err = fs.Write("mapper", "test_esb.py", TEST_SCRIPT_ESB); err != nil {
		t.Error(err)

	}
	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("mapper", "test_esb.py", TEST_ARGS_ESB)
		if err != nil {
			t.Error(err)
			return
		}

		if !bytes.Equal(out, []byte(EXPECTED_OUTPUT_ESB)) {
			t.Errorf(`%s != %s`, string(out), EXPECTED_OUTPUT_ESB)
		}

		worker.Return()
	}
}
