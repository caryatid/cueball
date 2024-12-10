// state package using internal memory
package state

import (
	"context"
	"cueball"
	"github.com/google/uuid"
)

var queue_size = 10 // TODO


type Mem struct {
	*Op
	queue chan cueball.Worker
	ids map[string]cueball.Worker
}

func NewMem(ctx context.Context) (*Mem, error) {
	s := new(Mem)
	s.Op = NewOp()
	s.queue = make(chan cueball.Worker, queue_size)
	s.ids = make(map[string]cueball.Worker)
	return s, nil
}

func (s *Mem) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	w, ok := s.ids[uuid.String()]
	if !ok {
		return nil, nil // TODO
	}
	return w, nil
}

func (s *Mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids[w.ID().String()] = w
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.queue <- w
	return nil
}

func (s *Mem) Dequeue(ctx context.Context, w cueball.Worker) error {
	ww := <- s.queue
	ww.FuncInit() // TODO not necessary for mem; do anyway as this is demonstrative
	s.Channel() <- ww
	return nil
}

func (s *Mem) LoadWork(ctx context.Context, w cueball.Worker) error {
	for _, w := range s.ids {
		if w.Stage() == cueball.RETRY || w.Stage() == cueball.INIT ||
			w.Stage() == cueball.NEXT {
			w.SetStage(cueball.RUNNING)
			s.Persist(ctx, w)
			if err := s.Enqueue(ctx, w); err != nil {
				return err
			}
		}
	}
	return nil
}

