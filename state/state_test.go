package state

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/retry"
)

type testWorker struct {
	cueball.Executor
	testString   string
	testInt      int
	testFloat    float64
	testStruct   nestData
	testStructPt *nestData
	testList     []string
}

type nestData struct {
	name  string
	class string
}

func NewTestWorker() cueball.Worker {
	sw := new(testWorker)
	var ss []cueball.Method
	sw.Executor = NewExecutor(retry.NewCount(3, ss...)...)
	return sw
}

func (w *testWorker) Name() string {
	return "test-worker"
}

// NOTE: do not use state so tests have empty state components
func (w *testWorker) Stage1(ctx context.Context, s cueball.State) error {
	w.testString = "one"
	w.testInt = 1
	w.testFloat = 1.0001
	w.testStruct = nestData{name: "zork", class: "wizard"}
	w.testStructPt = &nestData{name: "guts", class: "warrior"}
	w.testList = []string{"aaa", "bbb"}
	return nil
}

func (w *testWorker) Stage2(ctx context.Context, s cueball.State) error {
	w.testString = "two"
	w.testInt = 2
	w.testFloat = 2.0002
	w.testStruct = nestData{name: "cloud", class: "fighter"}
	w.testStructPt = &nestData{name: "grue", class: "monster"}
	w.testList = []string{"ccc", "ddd", "eee"}
	return nil
}
