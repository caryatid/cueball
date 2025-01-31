// Package state manages, you guessed it, state.
package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"time"
	"sync"
)

type defState struct {
	scmutex sync.Locker
	cueball.Pipe
	cueball.Record
	cueball.Blob
	g *errgroup.Group
	enq chan cueball.Worker
	deq chan cueball.Worker
	rec chan cueball.Worker
}

func NewState(ctx context.Context, p cueball.Pipe, r cueball.Record,
	b cueball.Blob) (cueball.State, context.Context) {
	s := new(defState)
	s.Pipe = p
	s.Record = r
	s.Blob = b
	s.scmutex = new(sync.Mutex)
	s.enq = make(chan cueball.Worker)
	s.deq = make(chan cueball.Worker)
	s.rec = make(chan cueball.Worker)
	s.g, ctx = errgroup.WithContext(ctx)
	return s, ctx
}

func (s *defState) runScan (ctx context.Context) chan cueball.Worker {
	ch := make(chan cueball.Worker)
	s.g.Go(func () error {
		s.scmutex.Lock()
		defer s.scmutex.Unlock()
		defer close(ch)
		return s.Scan(ctx, ch)
	})
	return ch
}

func (s *defState) Start(ctx context.Context) chan cueball.Worker {
	s.g.Go(func () error { return s.Dequeue(ctx, s.deq) })
	s.g.Go(func () error { return s.Enqueue(ctx, s.enq) })
	s.g.Go(func () error { return s.Store(ctx, s.rec) })
	s.g.Go(func () error {
		for {
			for w := range s.runScan(ctx, s.Scan) {
				w.SetStatus(cueball.INFLIGHT)
				s.rec <- w
				s.enq <- w
			}
			time.Sleep(time.Millisecond*15) // TODO
		}
		return nil
	})
	s.g.Go(func() error {
		for w := range s.deq {
			w.Do(ctx, s) // error handled inside
			if cueball.DirectEnqueue && !w.Done() {
				enq <- w
			}
			s.rec <- w
		}
		return nil
	})
	return enq
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
			s.Close()
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

func (s *defState) Close() error {
	if s.Blob != nil {
		//	s.Blob.Close()
	}
	if s.Pipe != nil {
		s.Pipe.Close()
	}
	if s.Record != nil {
		s.Record.Close()
	}
	return nil
}

// Pack facilitates multiplexing on an untyped queue
type Pack struct {
	Name  string
	Codec string
}

func Unmarshal(data string, w interface{}) error {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, w)
}

func Marshal(w interface{}) ([]byte, error) {
	b, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	data := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(data, b)
	return data, nil
}
