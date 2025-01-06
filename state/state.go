// Package cueball/state defines the [Operator] type which drives
// different backing states for cueball. A few State implementations
// are also provided.
package state

import (
	"context"
	"github.com/caryatid/cueball"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

// Operator, as the name suggests, does the do.
type Operator struct {
	sync.Mutex
	state cueball.State
	store chan cueball.Worker
}

// The Operator type needs your state implementation
func NewOperator(ctx context.Context, s cueball.State) *Operator {
	op := new(Operator)
	op.state = s
	op.store = make(chan cueball.Worker, cueball.ChanSize)
	return op
}

// Kicks off operation. Returns the errgroup and context parent of all
// goroutines started by cueball.
//
// # Multiple process considerations
//
// LoadWork is sequenced here for convenience. Additional locking will be required
// for most state implementations if there are multiple cueball processes
func (o *Operator) Start(ctx_ context.Context) (g *errgroup.Group, ctx context.Context) {
	g, ctx = errgroup.WithContext(ctx_)
	t := time.NewTicker(time.Millisecond * 25)
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				g.Go(func() error {
					o.Lock()
					defer o.Unlock()
					return o.state.LoadWork(ctx)
				})
			case w := <-o.store:
				if err := o.state.Persist(ctx, w); err != nil {
					return err
				}
			case w := <-o.state.Out():
				g.Go(func() error {
					w.StageInit()
					o.runstage(ctx, w)
					return nil
				})
			}
		}
	})
	return
}

// TODO could inline if this func stays small
func (o *Operator) runstage(ctx context.Context, w cueball.Worker) {
	s := w.Current()
	err := s.Do(ctx)
	if err != nil && s.Tries() >= cueball.MaxRetries {
		cueball.Lc(ctx).Debug().Err(err).Msg(cueball.FAIL.String())
		w.SetStatus(cueball.FAIL)
	} else if s.Done() {
		w.SetStatus(cueball.DONE)
	} else {
		if cueball.DirectEnqueue {
			o.state.Enqueue(ctx, w)
		} else {
			w.SetStatus(cueball.ENQUEUE)
		}
	}
	// cueball.Lc(ctx).Debug().Interface("W", w).Send()
	o.store <- w
}
