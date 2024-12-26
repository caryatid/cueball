package state

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"sync"
)

var queue_size = 10 // TODO

type Mem struct {
	cueball.WorkerSet
	sync.Mutex
	queue chan cueball.Worker
	ids   sync.Map
}

func NewMem(ctx context.Context, w ...cueball.Worker) (*Mem, error) {
	s := new(Mem)
	s.WorkerSet = DefaultWorkerSet()
	s.AddWorker(ctx, w...)
	s.queue = make(chan cueball.Worker, queue_size)
	go s.dequeue(ctx)
	return s, nil
}

func (s *Mem) emulateSerialize(w cueball.Worker) cueball.Worker {
	s.Lock()
	defer s.Unlock()
	b, _ := marshal(w)
	ww := w.New()
	unmarshal(string(b), ww)
	return ww
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
	return s.emulateSerialize(w), nil
}

func (s *Mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids.Store(w.ID().String(), w)
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.queue <- w
	return nil
}

func (s *Mem) dequeue(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case w := <-s.queue:
			s.Out() <- w
		}
	}
}

func (s *Mem) Close() error {
	return nil
}

func (s *Mem) LoadWork(ctx context.Context, ch chan cueball.Worker) error {
	s.ids.Range(func(k, w_ any) bool {
		w, _ := w_.(cueball.Worker)
		if w.Status() == cueball.RETRY ||
			w.Status() == cueball.INIT ||
			w.Status() == cueball.NEXT {
			ch <- s.emulateSerialize(w)
		}
		return true
	})
	return nil
}
