package worker

import (
	"context"
	"cueball"
)

type CountWorker struct {
	*Exec
	Cnt int
}

func (s *CountWorker) Name() string {
	return "count-worker"
}

func (s *CountWorker) FuncInit() {
	var x []cueball.Method
	for i:=0; i<10; i++ {
		x = append(x, s.Inc)
	}
	s.Load(x...)
}

func (s *CountWorker) New() cueball.Worker {
	sw := &CountWorker{Exec: NewExec()}
	return sw
}

func (s *CountWorker) Inc(ctx context.Context) error {
	s.Cnt += 10 
	return nil
}
