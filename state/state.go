package state

import (
	"cueball"
	"golang.org/x/sync/errgroup"
)

var ChanSize = 1 // Configuration? Argument to NewOp?

type Op struct {
	group *errgroup.Group
	workers []cueball.Worker
	intake chan cueball.Worker
}

func NewOp(g *errgroup.Group, w... cueball.Worker) (op *Op) {
	op.Load(w...) 
	op.intake = make(chan cueball.Worker, ChanSize)
	op.group = g 
	return 
}

func (o *Op) Load(w... cueball.Worker) {
	o.workers = append(o.workers, w...)
}

func (o *Op) Group() *errgroup.Group {
	return o.group
}

func (o *Op) Channel() chan cueball.Worker {
	return o.intake 
}

func (o *Op) Workers() []cueball.Worker {
	return o.workers
}

