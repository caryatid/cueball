package state

import (
	"context"
	"github.com/caryatid/cueball"
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
	return ws.out
}

func (ws *defaultWorkerSet) AddWorker(ctx context.Context, w cueball.Worker) {
	ws.workers.Store(w.Name(), w)
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
