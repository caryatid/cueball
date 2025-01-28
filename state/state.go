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

func (s *defState) Run(ctx context.Context, f cueball.RunFunc) chan cueball.Worker {
	ch := make(chan cueball.Worker)
	s.g.Go(func() error {
		return f(ctx, ch)
	})
	return ch
}

func (s *defState) Start(ctx context.Context) chan cueball.Worker {
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	store := s.Run(ctx, s.Store)
	s.g.Go(func() error {
		for {
			for w := range s.Run(ctx, s.Scan) {
				enq <- w
			}
		}
		return nil
	})
	s.g.Go(func() error {
		for w := range deq {
			w.Do(ctx, s) // error handled inside
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
	return enq
}

func (s *defState) Check(ctx context.Context, ids []uuid.UUID) bool {
	for _, id := range ids {
		w, err := s.Get(ctx, id)
		if err != nil || w == nil { // FIX: timing issue sidestepped.
			continue
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
				case <-time.After(wait * 4): // IDK
				}
				return nil
			}
		}
	}
}

func (s *defState) Close() error {
	if s.Pipe != nil {
		s.Pipe.Close()
	}
	if s.Log != nil {
		s.Log.Close()
	}
	if s.Blob != nil {
		//	s.Blob.Close()
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
