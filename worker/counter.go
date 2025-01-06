package worker

import (
	"context"
	"github.com/caryatid/cueball"
)

type CountWorker struct {
	cueball.Executor
	Cnt int
}

func (s *CountWorker) Name() string {
	return "count-worker"
}

func (s *CountWorker) StageInit() {
	var ss []cueball.Method
	for i := 0; i < 10; i++ {
		ss = append(ss, s.Inc)
	}
	s.Load(ss...)
}

func (s *CountWorker) New() cueball.Worker {
	sw := &CountWorker{Executor: NewExecutor(s.Name())}
	return sw
}

func (s *CountWorker) Inc(ctx context.Context) error {
	s.Cnt += 10
	if s.Cnt >= 100 {
		cueball.Lc(ctx).Debug().Int("value", s.Cnt).Msg("at or over 100")
	}
	return nil
}
