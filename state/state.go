package state

import (
	"cueball"
	"sync"
)

var ChanSize = 1 // Configuration? Argument to NewOp?

type Op struct {
	sync.Mutex
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
}

func NewOp() (op *Op) {
	op = new(Op)
	op.intake = make(chan cueball.Worker, ChanSize)
	op.workers = make(map[string]cueball.Worker)
	return
}

func (o *Op) Load(w cueball.Worker) {
	o.workers[w.Name()] = w
}

func (o *Op) Channel() chan cueball.Worker {
	return o.intake
}

func (o *Op) Workers() map[string]cueball.Worker {
	return o.workers
}
