// state package using internal memory
package state

import (
	"context"
	"cueball"
	"github.com/google/uuid"
	"sync"
)

var queue_size = 10 // TODO

type Mem struct {
	*Operator
	queue chan cueball.Worker
	ids   sync.Map
}

func NewMem(ctx context.Context, w ...cueball.Worker) (*Mem, error) {
	s := new(Mem)
	s.Operator = NewOperator(w...)
	s.queue = make(chan cueball.Worker, queue_size)
	return s, nil
}

func (s *Mem) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	w_, ok := s.ids.Load(uuid.String())
	if !ok {
		return nil, nil // TODO must error
	}
	w, ok := w_.(cueball.Worker)
	if !ok {
		return nil, nil // TODO error
	}
	return w, nil
}

func (s *Mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids.Store(w.ID().String(), w)
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.queue <- w
	return nil
}

func (s *Mem) Dequeue(ctx context.Context) error {
	w := <-s.queue
	s.Work() <- w
	return nil
}

func (s *Mem) Close() error {
	return nil
}

func (s *Mem) LoadWork(ctx context.Context) error {
	s.ids.Range(func(k, w_ any) bool {
		w, _ := w_.(cueball.Worker)
		if w.Stage() == cueball.RETRY || w.Stage() == cueball.INIT ||
			w.Stage() == cueball.NEXT {
			s.Intake() <- w
		}
		return true
	})
	return nil
}
