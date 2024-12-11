package state

import (
	"context"
	"cueball"
	"sync"
)

type defaultWorkerSet struct {
	workers sync.Map
	out     chan cueball.Worker
}

func DefaultWorkerSet() cueball.WorkerSet {
	ws := new(defaultWorkerSet)
	ws.out = make(chan cueball.Worker, cueball.ChanSize)
	return ws
}

func (ws *defaultWorkerSet) Out() chan cueball.Worker {
	if ws.out == nil {
	}
	return ws.out
}

func (ws *defaultWorkerSet) AddWorker(ctx context.Context, w ...cueball.Worker) error {
	for _, w_ := range w {
		ws.workers.Store(w_.Name(), w_)
	}
	return nil
}

func (ws *defaultWorkerSet) ByName(name string) cueball.Worker {
	w, ok := ws.workers.Load(name)
	if !ok {
		// TODO
		return nil
	}
	return w.(cueball.Worker)
}

// TODO implement as a generic `iter`
func (ws *defaultWorkerSet) List() []cueball.Worker {
	var x []cueball.Worker
	ws.workers.Range(func(_, v any) bool {
		x = append(x, v.(cueball.Worker))
		return true
	})
	return x
}
