// state package using internal memory
package state

import (
	"context"
	"cueball"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
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
	s.queue = make(chan Worker, queue_size)
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
	// NOTE could factor out the persists b/c using pointers
	// s.ids[w.ID().String()] = w
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	w.Stage = cueball.RUNNING
	err := s.Persist(ctx, w)
	s.queue <- w
	return err
}

func (s *Mem) Dequeue(ctx context.Context, w cueball.Worker) error {
	var err error
	ww := <- s.queue
	ww.FuncInit()
	s.Channel() <- ww
	return nil
}

func (s *Mem) LoadWork(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	for name, w := range s.ids {
		if w.Stage == cueball.RETRY || w.Stage == cueball.INIT ||
			w.Stage == cueball.NEXT {
			if err := s.Enqueue(ctx, w); err != nil {
				return err
			}
		}
	}
	return nil
}
