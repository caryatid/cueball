// Package cueball/state defines the [Operator] type which drives
// different backing states for cueball. A few State implementations
// are also provided.
package state

import (
	"context"
	"cueball"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

// Operator, as the name suggests, does the do.
type Operator struct {
	sync.Mutex
	state  cueball.State
	intake chan cueball.Worker
	store  chan cueball.Worker
}

// The Operator type needs your state implementation
func NewOperator(ctx context.Context, s cueball.State) *Operator {
	op := new(Operator)
	op.state = s
	op.intake = make(chan cueball.Worker, cueball.ChanSize)
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
					return o.state.LoadWork(ctx, o.intake)
				})
			case w := <-o.intake:
				g.Go(func() error {
					w.SetStatus(cueball.ENQUEUE)
					o.store <- w
					return o.state.Enqueue(ctx, w)
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

// Adds new workers to the queue and returns their id's
// the new argument determines if the passed workers, with
// their state, are enqueued, or if new workers of the
// concrete type passed should be enqueued.
func (o *Operator) Enqueue(new bool, ww ...cueball.Worker) (uid []uuid.UUID) {
	for _, w := range ww {
		wi := w
		if new {
			wi = w.New()
		}
		uid = append(uid, wi.ID())
		go func() { o.intake <- wi }()
	}
	return
}

// Works in concert with Enqueue. Reports if given id's are done working.
func (o *Operator) Done(ctx context.Context, id ...uuid.UUID) bool {
	for _, uid := range id {
		w, _ := o.state.Get(ctx, uid)
		if w == nil {
			return false
		}
		if w.Status() != cueball.DONE && w.Status() != cueball.FAIL {
			return false
		}
	}
	return true
}

func (o *Operator) runstage(ctx context.Context, w cueball.Worker) {
	err := w.Next(ctx)
	if err != nil && errors.Is(err, cueball.EndError) {
		w.SetStatus(cueball.DONE)
		o.store <- w
		return
	} else if err != nil {
		if w.Retry() {
			if true { // TODO option to bypass persistence
				w.SetStatus(cueball.RETRY)
				o.store <- w
				return
			}
			o.intake <- w
			return
		}
		w.SetStatus(cueball.FAIL)
		o.store <- w
		return
	}
	w.SetStatus(cueball.NEXT)
	o.store <- w
	return
}
