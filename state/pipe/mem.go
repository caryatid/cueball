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

func (p *mem) emulateSerialize(src, target cueball.Worker) error {
	b, _ := state.Marshal(src)
	return state.Unmarshal(string(b), target)
}

func (p *mem) Enqueue(ctx context.Context, ch chan cueball.Worker) error {
	for w := range ch {
		p.queue <- w
	}
	return nil
}

func (p *mem) Dequeue(ctx context.Context, ch chan cueball.Worker) error {
	defer close(ch)
	for w := range p.queue {
		w_ := cueball.Gen(w.Name())
		p.emulateSerialize(w, w_)
		ch <- w_
	}
	return nil
}
