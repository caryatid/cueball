// state package using internal memory
package state

import (
	"context"
	"cueball"
	"github.com/google/uuid"
)

var queue_size = 10 // TODO


type Mem struct {
	queue chan cueball.Worker
	ids map[string]cueball.Worker
}

func NewMem(ctx context.Context) (*Mem, error) {
	s := new(Mem)
	s.queue = make(chan cueball.Worker, queue_size)
	s.ids = make(map[string]cueball.Worker)
	return s, nil
}

func (s *Mem) Get(ctx context.Context, w cueball.Worker, uuid uuid.UUID) error {
	var ok bool
	w, ok = s.ids[uuid.String()]
	if !ok {
		return nil // TODO must error
	}
	return nil
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
	w = <- s.queue
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

