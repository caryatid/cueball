package pipe

import (
	"context"
	"encoding/json"
	"github.com/caryatid/cueball"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

// pub/sub key prefix for nats
var prefix = "cueball.pg."

type natsp struct {
	Nats *nats.Conn
	sub  sync.Map
	g    *errgroup.Group
}

func NewNats(ctx context.Context, natsurl string) (cueball.Pipe, error) {
	var err error
	p := new(natsp)
	p.g, ctx = errgroup.WithContext(ctx)
	p.Nats, err = nats.Connect(natsurl)
	if err != nil {
		return nil, err
	}
	return p, err
}

func (p *natsp) Close() error {
	p.sub.Range(func(k, sub_ any) bool {
		sub := sub_.(*nats.Subscription)
		sub.Drain()
		return true
	})
	p.Nats.Drain()
	return nil
}

func (p *natsp) Enqueue(ctx context.Context, ch chan cueball.Worker) error {
	for w := range ch {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, err := json.Marshal(w)
		if err != nil {
			return err
		}
		if err := p.Nats.Publish(prefix+w.Name(), data); err != nil {
			return err
		}
	}
	return nil
}

func (p *natsp) subread(ctx context.Context, name string,
	ch chan cueball.Worker) error {
	subname := prefix + name
	sub, err := p.Nats.QueueSubscribeSync(subname, subname)
	if err != nil {
		return err
	}
	p.sub.Store(name, sub)
	p.g.Go(func() error {
		for {
			msg, err := sub.NextMsgWithContext(ctx)
			if err != nil {
				return err
			}
			w := cueball.Gen(name)
			if err := json.Unmarshal(msg.Data, w); err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			ch <- w
		}
	})
	return nil
}

func (p *natsp) workerscan(ctx context.Context, ch chan cueball.Worker) {
	for _, name := range cueball.Workers() {
		sub_, ok := p.sub.Load(name)
		if !ok || !sub_.(*nats.Subscription).IsValid() {
			if ok {
				sub_.(*nats.Subscription).Unsubscribe()
				sub_.(*nats.Subscription).Drain()
			}
			p.subread(ctx, name, ch)
		}
	}
}

func (p *natsp) Dequeue(ctx context.Context, ch chan cueball.Worker) error {
	defer close(ch)
	t := time.NewTicker(time.Millisecond * 150)
	p.workerscan(ctx, ch)
	for {
		select {
		case <-ctx.Done():
			return p.g.Wait()
		case <-t.C:
			p.workerscan(ctx, ch)
		}
	}
	return nil
}
