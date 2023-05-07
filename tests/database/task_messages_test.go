package database_test

import (
	"context"
	"mock-server/internal/control"
	"mock-server/internal/database"
	"reflect"
	"testing"
)

func TestTaskMessages(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/configs/test_database_config.yaml")
	control.Components.Start()
	defer control.Components.Stop()

	expectedTask1Messages := []database.TaskMessage{
		{TaskId: "task1", Message: "one"},
		{TaskId: "task1", Message: "two"},
	}
	expectedTask2Messages := []database.TaskMessage{
		{TaskId: "task2", Message: "three"},
		{TaskId: "task2", Message: "four"},
		{TaskId: "task2", Message: "five"},
	}

	for _, msg := range expectedTask1Messages {
		if err := database.AddTaskMessage(context.TODO(), msg); err != nil {
			t.Error(err)
		}
	}

	for _, msg := range expectedTask2Messages {
		if err := database.AddTaskMessage(context.TODO(), msg); err != nil {
			t.Error(err)
		}
	}

	actualTask1Messages, err := database.GetTaskMessages(context.TODO(), "task1")
	if err != nil {
		t.Error(err)
	}
	if reflect.DeepEqual(expectedTask1Messages, actualTask1Messages) {
		t.Errorf("res != expected: %+q != %+q", actualTask1Messages, expectedTask1Messages)
	}

	actualTask2Messages, err := database.GetTaskMessages(context.TODO(), "task2")
	if err != nil {
		t.Error(err)
	}
	if reflect.DeepEqual(expectedTask2Messages, actualTask2Messages) {
		t.Errorf("res != expected: %+q != %+q", actualTask2Messages, expectedTask2Messages)
	}
}
