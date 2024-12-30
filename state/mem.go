package state

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"sync"
	"time"
)

var queue_size = 10 // TODO

type mem struct {
	sync.Mutex
	cueball.WorkerSet
	queue chan cueball.Worker
	ids   sync.Map
}

func NewMem(ctx context.Context, works ...cueball.WorkerGen) (cueball.State, error) {
	s := new(mem)
	s.WorkerSet = DefaultWorkerSet(works...)
	s.queue = make(chan cueball.Worker, queue_size)
	go s.dequeue(ctx)
	return s, nil
}

func (s *mem) emulateSerialize(src, target cueball.Worker) error {
	b, _ := marshal(src)
	return unmarshal(string(b), target)
}

func (s *mem) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	w_, ok := s.ids.Load(uuid.String())
	if !ok {
		return nil, nil // TODO must error
	}
	w__, ok := w_.(cueball.Worker)
	if !ok {
		return nil, nil // TODO error
	}
	w := s.NewWorker(w__.Name())
	s.emulateSerialize(w__, w)
	return w, nil
}

func (s *mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids.Store(w.ID().String(), w)
	return nil
}

func (s *mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.queue <- w
	return nil
}

func (s *mem) dequeue(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case w := <-s.queue:
			w_ := s.NewWorker(w.Name())
			s.emulateSerialize(w, w_)
			s.Work() <- w_
		}
	}
}

func (s *mem) Close() error {
	return nil
}

func (s *mem) LoadWork(ctx context.Context) error {
	s.ids.Range(func(k, w_ any) bool {
		w, _ := w_.(cueball.Worker)
		if w.Status() == cueball.ENQUEUE && w.GetDefer().Before(time.Now()) {
			w.SetStatus(cueball.INFLIGHT)
			s.Enqueue(ctx, w)
		}
		return true
	})
	return nil
}
