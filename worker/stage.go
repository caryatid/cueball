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

func NewStageWorker() *stageWorker {
	sw := new(stageWorker)
	sw.Executor = NewExecutor(retry.NewCount(3, sw.Stage1, sw.Stage2, sw.Stage3)...)
	return sw
}

func (s *stageWorker) Stage1(ctx context.Context) error {
	s.Number = rand.Int() % 10
	s.Printer(ctx)
	return nil
}

func (s *stageWorker) Stage2(ctx context.Context) error {
	s.Printer(ctx)
	if s.Number < 4 {
		s.Number = rand.Int() % 10
		return fmt.Errorf("an error")
	}
	return nil
}

func (s *stageWorker) Stage3(ctx context.Context) error {
	s.Printer(ctx)
	return nil
}
