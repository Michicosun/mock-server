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

var TEST_SCRIPT_BAD_SCRIPT = util.WrapCodeForDynHandle(`
def func():
	print(
`)
var TEST_ARGS_BAD_SCRIPT = coderun.NewDynHandleArgs([]byte(`{}`))

func TestCoderunBadScript(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_coderun_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	if err = fs.Write("dyn_handle", "test_bad_script.py", TEST_SCRIPT_BAD_SCRIPT); err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("dyn_handle", "test_dyn_handle.py", TEST_ARGS_BAD_SCRIPT)
		t.Logf("Run output: %s", out)
		if err != coderun.ErrCodeRunFailed {
			t.Errorf("Expected `ErrCodeRunFailed` got: %s", err.Error())
			return
		}

		worker.Return()
	}
}

var TEST_SCRIPT_BAD_ARGS = util.WrapCodeForDynHandle(`
def func(a, b, c):
	print(c, b, a)
`)
var TEST_ARGS_BAD_ARGS = coderun.NewDynHandleArgs([]byte(`
{
	"a": 1,
	"b": 2,
	"d": 3
}`))

func TestCoderunBadArgs(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_coderun_config.yaml")

	control.Components.Start()
	defer control.Components.Stop()

	fs, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		t.Error(err)
	}

	if err = fs.Write("dyn_handle", "test_bad_script.py", TEST_SCRIPT_BAD_ARGS); err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("dyn_handle", "test_dyn_handle.py", TEST_ARGS_BAD_ARGS)
		t.Logf("Run output: %s", out)
		if err != coderun.ErrCodeRunFailed {
			t.Errorf("Expected `ErrCodeRunFailed` got: %s", err.Error())
			return
		}

		worker.Return()
	}
}
