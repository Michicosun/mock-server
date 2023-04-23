package coderun_test

import (
	"bytes"
	"mock-server/internal/coderun"
	"mock-server/internal/control"
	"mock-server/internal/util"
	"testing"
)

type ComplexArgs struct {
	A string   `json:"A"`
	B int      `json:"B"`
	C []string `json:"C"`
}

func TestCoderun(t *testing.T) {
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

	for i := 0; i < 10; i += 1 {
		worker, err := coderun.WorkerWatcher.BorrowWorker()
		if err != nil {
			t.Error(err)
		}

		out, err := worker.RunScript("mapper", "test.py", ComplexArgs{
			A: "sample_A",
			B: 42,
			C: []string{"a", "b", "c"},
		})
		if err != nil {
			t.Error(err)
			return
		}

		if bytes.Equal(out, []byte("Hello, world!")) {
			t.Errorf(`%s != "Hello, world!"`, string(out))
		}

		worker.Return()
	}
}
