package state

import (
	"context"
	"cueball"
	"sync"
	"golang.org/x/sync/errgroup"
	"errors"
	"time"
)

var ChanSize = 1 // Configuration? Argument to NewOperator?

type Operator struct {
	sync.Mutex
	state cueball.State
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
}

func NewOperator(s cueball.State, w ...cueball.Worker) *Operator {
	op := new(Operator)
	op.state = s
	op.intake = make(chan cueball.Worker, ChanSize)
	op.workers = make(map[string]cueball.Worker)
	op.Add(w...)
	return op
}

func (o *Operator) Add(w ...cueball.Worker) {
	for _, ww := range w {
		o.workers[ww.Name()] = ww
	}
}

func (o *Operator) Workers() map[string]cueball.Worker {
	return o.workers
}

func (o *Operator) Start(ctx context.Context) *errgroup.Group {
	g, ctx := errgroup.WithContext(ctx)
	for _, w := range o.Workers() {
		g.Go(func() error {
			return driver(ctx, w, o.loadWork)
		})
		g.Go(func() error {
			return driver(ctx, w, o.dequeue)
		})
	}
	for i := 0; i <= cueball.Worker_count; i++ {
		g.Go(func() error {
			for {
				select {
				case <- ctx.Done():
					return nil // TODO
				case w := <-o.intake:
					g.Go(func() error {
						return o.runstage(ctx, w)
					})
				}
			}
		})
	}
	return g
}

func (o *Operator) dequeue(ctx context.Context, w cueball.Worker) error {
	ww := w.New()
	err := o.state.Dequeue(ctx, ww)
	if err != nil && errors.Is(err, cueball.RequeueError) {
		return nil
	} else if err != nil {
		return err
	}
	ww.FuncInit()
	ww.SetStage(cueball.RUNNING)
	o.intake <- ww
	return o.state.Persist(ctx, ww)
}

func (o *Operator) enqueue(ctx context.Context, w cueball.Worker) error {
	o.Lock()
	defer o.Unlock()
	w.SetStage(cueball.ENQUEUE)
	if err := o.state.Enqueue(ctx, w); err != nil {
		return err
	}
	return o.state.Persist(ctx, w)
}

func (o *Operator) loadWork(ctx context.Context, w cueball.Worker) error {
	g, ctx := errgroup.WithContext(ctx)	
	ch := make(chan cueball.Worker)
	g.Go(func () error {
		defer close(ch)
		return o.state.LoadWork(ctx, w, ch)
	})
	g.Go(func () error {
		for {
			select {
			case <- ctx.Done():
				return nil // TODO error?
			case ww, ok := <- ch:
				if !ok {
					return nil	
				}
				if err := o.enqueue(ctx, ww); err != nil {
					return err
				}
			}
		}
	})
	return g.Wait()

}


func (o *Operator)runstage(ctx context.Context, w cueball.Worker) error {
	// TODO option allowing all stages on one thread?
	log := cueball.Lc(ctx).With().Str("name", w.Name()).Logger()
	err := w.Next(ctx)
	if err != nil && errors.Is(err, cueball.EndError) {
		w.SetStage(cueball.DONE) 
		log.Debug().Interface("worker", w).Msg("Process Complete")
		return o.state.Persist(ctx, w) 
	} else if err != nil {
		if w.Retry() {
			if true { // TODO option to bypass persistence
				w.SetStage(cueball.RETRY)
				log.Debug().Err(err).Interface("worker", w).Msg("retry persist")
				return o.state.Persist(ctx, w)
			} 
			log.Debug().Err(err).Interface("worker", w).Msg("retry re-enqueue")
			return o.enqueue(ctx, w)
		} 
		w.SetStage(cueball.FAIL)
		log.Debug().Err(err).Interface("worker", w).Msg("fail")
		return o.state.Persist(ctx, w)
	} 
	w.SetStage(cueball.NEXT)
	log.Debug().Interface("worker", w).Msg("next stage")
	return o.state.Persist(ctx, w) 
}

func driver(ctx context.Context, w cueball.Worker, f cueball.WorkFunc) error {
	tick := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <- ctx.Done():
			return nil // TODO
		case <- tick.C:
			if err := f(ctx, w); err != nil {
				return err
			}
		}
	}
}

