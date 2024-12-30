package worker

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/retry"
)

type countWorker struct {
	cueball.Executor
	Cnt int
}

func NewCountWorker() cueball.Worker {
	sw := new(countWorker)
	var ss []cueball.Method
	for i := 0; i < 10; i++ {
		ss = append(ss, sw.Inc)
	}
	sw.Executor = NewExecutor(retry.NewCount(3, ss...)...)
	return sw
}

func (s *countWorker) Name() string {
	return "count-worker"
}

func (s *countWorker) Inc(ctx context.Context) error {
	s.Cnt += 10
	if s.Cnt >= 100 {
		cueball.Lc(ctx).Debug().Int("value", s.Cnt).Msg("at or over 100")
	}
	return nil
}
