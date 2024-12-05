package state

import (
	"cueball"
	"golang.org/x/sync/errgroup"
	"sync"
)

var ChanSize = 1 // Configuration? Argument to NewOp?

type Op struct {
	sync.Mutex
	group   *errgroup.Group
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
	init    func(*errgroup.Group, cueball.Worker)
}

func NewOp(g *errgroup.Group) (op *Op) {
	op.intake = make(chan cueball.Worker, ChanSize)
	op.group = g
	return
}

func (o *Op) Load(w cueball.Worker) {
	o.workers[w.Name()] = w
}

func (o *Op) Group() *errgroup.Group {
	return o.group
}

func (o *Op) Channel() chan cueball.Worker {
	return o.intake
}

func (o *Op) Workers() map[string]cueball.Worker {
	return o.workers
}

