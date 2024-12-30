package state

import (
	"github.com/caryatid/cueball"
)

type defaultWorkerSet struct {
	workers map[string]cueball.WorkerGen
	work    chan cueball.Worker
	store   chan cueball.Worker
}

func DefaultWorkerSet(works ...cueball.WorkerGen) cueball.WorkerSet {
	ws := new(defaultWorkerSet)
	ws.work = make(chan cueball.Worker, cueball.ChanSize)
	ws.store = make(chan cueball.Worker, cueball.ChanSize)
	ws.workers = make(map[string]cueball.WorkerGen)
	ws.AddWorker(works...)
	return ws
}

func (ws *defaultWorkerSet) Work() chan cueball.Worker {
	return ws.work
}

func (ws *defaultWorkerSet) Store() chan cueball.Worker {
	return ws.store
}

func (ws *defaultWorkerSet) AddWorker(works ...cueball.WorkerGen) {
	// NOTE: overwrites
	for _, w := range works {
		ws.workers[w().Name()] = w // NOTE: makes and discards a worker
	}
}

func (ws *defaultWorkerSet) NewWorker(name string) cueball.Worker {
	w, ok := ws.workers[name]
	if !ok {
		// TODO
		return nil
	}
	return w()
}

// TODO implement as a generic `iter`?
func (ws *defaultWorkerSet) Workers() map[string]cueball.WorkerGen {
	return ws.workers
}
