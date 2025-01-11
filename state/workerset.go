package state

import (
	"context"
	"github.com/caryatid/cueball"
	"golang.org/x/sync/errgroup"
)

type defaultWorkerSet struct {
	work  chan cueball.Worker
	store chan cueball.Worker
	g     *errgroup.Group
}

func DefaultWorkerSet(ctx context.Context) (cueball.WorkerSet, context.Context) {
	ws := new(defaultWorkerSet)
	ws.work = make(chan cueball.Worker, cueball.ChanSize)
	ws.store = make(chan cueball.Worker, cueball.ChanSize)
	ws.g, ctx = errgroup.WithContext(ctx)
	return ws, ctx
}

func (ws *defaultWorkerSet) Work() chan cueball.Worker {
	return ws.work
}

func (ws *defaultWorkerSet) Group() *errgroup.Group {
	return ws.g
}

func (ws *defaultWorkerSet) Store() chan cueball.Worker {
	return ws.store
}
