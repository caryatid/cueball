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

// func NewCountWorker() *countWorker {
func NewCountWorker() cueball.Worker {
	sw := new(countWorker)
	var ss []cueball.Method
	for i := 0; i < 10; i++ {
		ss = append(ss, sw.Inc)
	}
	sw.Executor = NewExecutor(retry.NewCount(3, ss...)...)
	return sw
}

func (w *countWorker) Name() string {
	return "count-worker"
}

func (w *countWorker) Inc(ctx context.Context, s cueball.State) error {
	w.Cnt += 10
	if w.Cnt >= 100 {
		cueball.Lc(ctx).Debug().Interface("W", w).Msg("at or over 100")
	}
	return nil
}
