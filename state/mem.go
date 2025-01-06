package state

import (
	"context"
	"github.com/caryatid/cueball"
	"sync"
)

var queue_size = 10 // TODO

type Mem struct {
	cueball.WorkerSet
	sync.Mutex
	queue chan cueball.Worker
	ids   sync.Map
}

func NewMem(ctx context.Context) (*Mem, error) {
	s := new(Mem)
	s.WorkerSet = DefaultWorkerSet()
	s.queue = make(chan cueball.Worker, queue_size)
	go s.dequeue(ctx)
	return s, nil
}

func (s *Mem) emulateSerialize(src, target cueball.Worker) error {
	b, _ := marshal(src)
	return unmarshal(string(b), target)
}

func (s *Mem) Get(ctx context.Context, w cueball.Worker) error {
	w_, ok := s.ids.Load(w.ID().String())
	if !ok {
		return nil // TODO must error
	}
	w__, ok := w_.(cueball.Worker)
	if !ok {
		return nil // TODO error
	}
	s.emulateSerialize(w__, w)
	w.StageInit()
	return nil
}

func (s *Mem) Persist(ctx context.Context, w cueball.Worker) error {
	s.ids.Store(w.ID().String(), w)
	return nil
}

func (s *Mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.AddWorker(ctx, w)
	s.queue <- w
	return nil
}

func (s *Mem) dequeue(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case w := <-s.queue:
			w_ := w.New()
			s.emulateSerialize(w, w_)
			s.Out() <- w_
		}
	}
}

func (s *Mem) Close() error {
	return nil
}

func (s *Mem) LoadWork(ctx context.Context) error {
	s.ids.Range(func(k, w_ any) bool {
		w, _ := w_.(cueball.Worker)
		if w.Status() == cueball.ENQUEUE {
			s.Enqueue(ctx, w)
		}
		return true
	})
	return nil
}
