package worker

import (
	"context"
	"fmt"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/retry"
	"math/rand"
)

type stageWorker struct {
	cueball.Executor
	Word   string
	Number int
}

func (s *stageWorker) Name() string {
	return "stage-worker"
}

func (s *stageWorker) Printer(ctx context.Context) {
	//log := cueball.Lc(ctx)
	// log.Debug().Interface("worker", s).Msg("from stage worker")
}

// func NewStageWorker() *stageWorker {
func NewStageWorker() cueball.Worker {
	sw := new(stageWorker)
	sw.Executor = NewExecutor(retry.NewCount(3, sw.Stage1, sw.Stage2, sw.Stage3)...)
	return sw
}

func (w *stageWorker) Stage1(ctx context.Context, s cueball.State) error {
	w.Number = rand.Int() % 10
	w.Printer(ctx)
	return nil
}

func (w *stageWorker) Stage2(ctx context.Context, s cueball.State) error {
	w.Printer(ctx)
	if w.Number < 4 {
		w.Number = rand.Int() % 10
		return fmt.Errorf("an error")
	}
	return nil
}

func (w *stageWorker) Stage3(ctx context.Context, s cueball.State) error {
	w.Printer(ctx)
	return nil
}
