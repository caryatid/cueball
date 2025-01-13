package worker

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/retry"
)

type testWorker struct {
	cueball.Executor
	TestString   string
	TestInt      int
	TestFloat    float64
	TestStruct   nestData
	TestStructPt *nestData
	TestList     []string
}

type nestData struct {
	Name  string
	Class string
}

func NewTestWorker() cueball.Worker {
	sw := new(testWorker)
	sw.Executor = NewExecutor(retry.NewCount(3, sw.Stage1, sw.Stage2)...)
	return sw
}

func (w *testWorker) Name() string {
	return "test-worker"
}

// NOTE: do not use state so tests may have nil state args
func (w *testWorker) Stage1(ctx context.Context, s cueball.State) error {
	w.TestString = "one"
	w.TestInt = 1
	w.TestFloat = 1.0001
	w.TestStruct = nestData{Name: "zork", Class: "wizard"}
	w.TestStructPt = &nestData{Name: "guts", Class: "warrior"}
	w.TestList = []string{"aaa", "bbb"}
	return nil
}

func (w *testWorker) Stage2(ctx context.Context, s cueball.State) error {
	w.TestString = "two"
	w.TestInt = 2
	w.TestFloat = 2.0002
	w.TestStruct = nestData{Name: "cloud", Class: "fighter"}
	w.TestStructPt = &nestData{Name: "grue", Class: "monster"}
	w.TestList = []string{"ccc", "ddd", "eee"}
	return nil
}
