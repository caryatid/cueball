// Package cueball/state

package state

import (
	"context"
	"github.com/caryatid/cueball"
	"golang.org/x/sync/errgroup"
	"time"
)

type defState struct {
	cueball.Pipe
	cueball.Log
	cueball.Blob
	g *errgroup.Group
}

func NewState(ctx context.Context, p cueball.Pipe, l cueball.Log,
		b cueball.Blob) (cueball.State, context.Context) {
	s := new(defState)
	s.Pipe = p
	s.Log = l
	s.Blob = b
	s.g, ctx = errgroup.WithContext(ctx)
	return s, ctx
}

func (s *defState)run(ctx context.Context, f cueball.RunFunc) chan cueball.Worker {
	ch := make(chan cueball.Worker)
	s.g.Go(func () error {
		return f(ctx, ch)
	})
	return ch
}

func (s *defState) Start(ctx context.Context) {
	enq := s.run(ctx, s.Enqueue)
	deq := s.run(ctx, s.Dequeue)
	store := s.run(ctx, s.Store)
	s.g.Go(func() error {
		for {
			for w := range s.Run(ctx, s.Scan) {
				w.SetStatus(cueball.INFLIGHT)
				store <- w
				enq <- w
			}
		}
		return nil
	})
	s.g.Go(func() error {
		for {
			w := <- deq
			w.Do(ctx) // error handled inside
			if !w.Done() {
				if cueball.DirectEnqueue {
					enq <- w
				} else {
					w.SetStatus(cueball.ENQUEUE)
				}
			}
			store <- w
		}
		return nil
	})
	return
}

func (s *defState)Wait(ctx context.Context, wait time.Duration, ids []uuid.UUID) error {
	t := time.NewTicker(wait)
	for {
		select {
		case <-ctx.Done():
			s.Close()
			return nil
		case <-tick.C:
			gtg := true
			for _, id := range ids {
				w, err := s.Get(ctx, id)
				if err != nil || w == nil {
					continue
				}
				if !w.Done() {
					gtg = false
					break
				}
			}
			if gtg {
				c := make(chan error)
				go func() {
					defer close(c)
					c <- s.Group().Wait()
				}()
				select {
				case err, _ := <-c: // TODO handle ok
					return err
				case <-time.After(time.Millisecond * 750):
				}
				return nil
			}
		}
	}
}
