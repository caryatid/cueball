package worker

import (
	"context"
	"fmt"
	"github.com/caryatid/cueball"
	"math/rand"
)

type StageWorker struct {
	cueball.Executor
	Word   string
	Number int
}

func (s *StageWorker) Name() string {
	return "stage-worker"
}

func (s *StageWorker) StageInit() {
	s.Load(s.Stage1, s.Stage2, s.Stage3)
}

func (s *StageWorker) Printer(ctx context.Context) {
	log := cueball.Lc(ctx)
	log.Debug().Interface("worker", s).Msg("from stage worker")
}

func (s *StageWorker) New() cueball.Worker {
	sw := &StageWorker{Executor: NewExecutor()}
	return sw
}

func (s *StageWorker) Stage1(ctx context.Context) error {
	s.Number = rand.Int() % 10
	s.Printer(ctx)
	return nil
}

func (s *StageWorker) Stage2(ctx context.Context) error {
	s.Printer(ctx)
	if s.Number < 4 {
		s.Number = rand.Int() % 10
		return fmt.Errorf("an error")
	}
	return nil
}

func (s *StageWorker) Stage3(ctx context.Context) error {
	s.Printer(ctx)
	return nil
}
