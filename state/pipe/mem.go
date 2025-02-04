package pipe

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
)

var queue_size = 1 // TODO

type mem struct {
	queue chan cueball.Worker
}

func NewMem(ctx context.Context) (cueball.Pipe, error) {
	p := new(mem)
	p.queue = make(chan cueball.Worker, queue_size)
	return p, nil
}

func (p *mem) Close() error {
	//	close(p.queue)
	return nil
}

func (p *mem) emulateSerialize(src, target cueball.Worker) error {
	b, _ := state.Marshal(src)
	return state.Unmarshal(string(b), target)
}

func (p *mem) Enqueue(ctx context.Context, w cueball.Worker) error {
	p.queue <- w
	return nil
}

func (p *mem) Dequeue(ctx context.Context, ch chan<- cueball.Worker) error {
	w_ := <-p.queue
	w := cueball.GenWorker(w_.Name())
	p.emulateSerialize(w_, w)
	ch <- w
	return nil
}
