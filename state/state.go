// Package state manages, you guessed it, state.
package state

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type defState struct {
	scmutex sync.Locker
	cueball.Blob
	p   cueball.Pipe
	r   cueball.Record
	g   *errgroup.Group
	enq chan cueball.Worker
	deq chan cueball.Worker
	rec chan cueball.Worker
}

func NewState(ctx context.Context, p cueball.Pipe, r cueball.Record,
	b cueball.Blob) (cueball.State, context.Context) {
	s := new(defState)
	s.Blob = b
	s.p = p
	s.r = r
	s.scmutex = new(sync.Mutex)
	s.enq = make(chan cueball.Worker)
	s.deq = make(chan cueball.Worker)
	s.rec = make(chan cueball.Worker)
	s.g, ctx = errgroup.WithContext(ctx)
	s.g.Go(func() error { return rangeRun(ctx, s.deq, s.Work) })
	if s.p != nil {
		s.g.Go(func() error { return rangeRun(ctx, s.enq, s.p.Enqueue) })
		s.g.Go(func() error { return tickRun(ctx, s.deq, nil, s.p.Dequeue) })
	}
	if s.r != nil {
		s.g.Go(func() error { return rangeRun(ctx, s.rec, s.r.Store) })
		s.g.Go(func() error { return tickRun(ctx, s.enq, s.scmutex, s.r.Scan) })
	}
	return s, ctx
}

func (s *defState) Work(ctx context.Context, w cueball.Worker) error {
	w.Do(ctx, s) // error handled inside
	if cueball.DirectEnqueue && !w.Done() {
		s.enq <- w
	}
	s.rec <- w
	return nil
}

func (s *defState) Get(ctx context.Context, u uuid.UUID) (cueball.Worker, error) {
	return s.r.Get(ctx, u)
}

func (s *defState) Check(ctx context.Context, ids []uuid.UUID) bool {
	for _, id := range ids {
		w, err := s.Get(ctx, id)
		if err != nil || w == nil { // FIX: timing issue sidestepped.
			return false
		}
		if !w.Done() {
			return false
		}
	}
	return true
}

func (s *defState) Wait(ctx context.Context, wait time.Duration, checks []uuid.UUID) error {
	tick := time.NewTicker(wait)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if s.Check(ctx, checks) {
				c := make(chan error)
				go func() {
					defer close(c)
					c <- s.g.Wait()
				}()
				select {
				case err, _ := <-c: // TODO handle ok
					return err
				case <-time.After(wait * 3): // IDK
				}
				return nil
			}
		}
	}
}

func (s *defState) Enq() chan<- cueball.Worker {
	return s.enq
}

func (s *defState) Close() error {
	if s.Blob != nil {
		//	s.Blob.Close()
	}
	if s.p != nil {
		s.p.Close()
	}
	if s.r != nil {
		s.r.Close()
	}
	return nil
}

func rangeRun(ctx context.Context, ch <-chan cueball.Worker,
	f cueball.WMethod) error {
	for w := range ch {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := f(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func tickRun(ctx context.Context, ch chan<- cueball.Worker, l sync.Locker,
	f cueball.WCMethod) error {
	t := time.NewTicker(time.Millisecond * 5)
	if err := lockRun(ctx, ch, l, f); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := lockRun(ctx, ch, l, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func lockRun(ctx context.Context, ch chan<- cueball.Worker, l sync.Locker,
	f cueball.WCMethod) error {
	if l != nil {
		l.Lock()
		defer l.Unlock()
	}
	return f(ctx, ch)
}
