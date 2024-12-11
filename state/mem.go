// state package using internal memory
package state

import (
	"sync"
	"context"
	"cueball"
	"github.com/google/uuid"
)

var queue_size = 10 // TODO


type Mem struct {
	queue chan Pack  // emulate a real persistence layer
	//ids map[string]cueball.Worker
	ids sync.Map

}

func NewMem(ctx context.Context) (*Mem, error) {
	s := new(Mem)
	s.queue = make(chan Pack, queue_size)
	return s, nil
}

func (s *Mem) Get(ctx context.Context, w cueball.Worker, uuid uuid.UUID) (cueball.Worker, error) {
	ww, ok := s.ids.Load(uuid.String())
	if !ok {
		return nil, nil // TODO must error
	}
	return ww, nil
}

func (s *Mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids.Store(w.ID().String(), w)
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := marshal(w)
	if err != nil {
		return err
	}
	p := Pack{Name: w.Name(), Codec: string(data)}
	s.queue <- p
	return nil
}

func (s *Mem) Dequeue(ctx context.Context, w cueball.Worker) error {
	p := <- s.queue
	if p.Name == w.Name() {
		return unmarshal(p.Codec, w)	
	}
	s.queue <- p
	return cueball.RequeueError
}

func (s *Mem) LoadWork(ctx context.Context, w cueball.Worker, ch chan cueball.Worker) error {
	s.ids.Range(func (k, www any) bool {
		ww, _ := www.(cueball.Worker)
		if ww.Stage() == cueball.RETRY || ww.Stage() == cueball.INIT ||
				ww.Stage() == cueball.NEXT {
			ch <- ww
		}
		return true
	})
	return nil
}

