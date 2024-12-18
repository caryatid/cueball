package state

import (
	"time"
	"context"
	"cueball"
	"errors"
	"golang.org/x/sync/errgroup"
	"sync"
)

var ChanSize = 1 // Configuration? Argument to NewOperator?
var retry_count = 3

type Operator struct {
	sync.Mutex
	state   cueball.State
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
	work    chan cueball.Worker
	store   chan cueball.Worker
}

func NewOperator(ctx context.Context, s cueball.State, w ...cueball.Worker) *Operator {
	op := new(Operator)
	op.state = s
	op.intake = make(chan cueball.Worker, ChanSize)
	op.work = make(chan cueball.Worker, ChanSize)
	op.store = make(chan cueball.Worker, ChanSize)
	op.workers = make(map[string]cueball.Worker)
	op.AddWorker(ctx, w...)
	return op
}

func (o *Operator) AddWorker(ctx context.Context, w ...cueball.Worker) {
	for _, ww := range w {
		o.workers[ww.Name()] = ww
		op.state.RegWorker(ctx, w)
	}
}

func (o *Operator) Work() chan cueball.Worker {
	return o.work
}

func (o *Operator) Intake() chan cueball.Worker {
	return o.intake
}

func (o *Operator) Store() chan cueball.Worker {
	return o.store
}

func (o *Operator) Workers() map[string]cueball.Worker {
	return o.workers
}

func (o *Operator) Start(ctx_ context.Context, s cueball.State) (g *errgroup.Group, ctx context.Context) {
	log := cueball.Lc(ctx_)
	g, ctx = errgroup.WithContext(ctx_)
	// backgrounders
	g.Go(func() error {
		t := time.NewTicker(time.Millisecond * 15)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := s.Dequeue(ctx, g); err != nil {
					return err
				}
			}
		}
		return nil
	})
	g.Go(func() error {
		t := time.NewTicker(time.Millisecond * 150)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := s.LoadWork(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case w := <-o.intake:
				w.SetStage(cueball.ENQUEUE)
				if err := s.Enqueue(ctx, w); err != nil {
					log.Debug().Err(err).Msg("Enqueue error")
					return err
				}
				g.Go(func () error { o.store <- w; return nil })
			case w := <-o.store:
				if err := s.Persist(ctx, w); err != nil {
					log.Debug().Err(err).Msg("Persist error")
					return err
				}
			case w := <-o.work:
				w.FuncInit()
				g.Go(func () error { o.runstage(ctx, s, w); return nil })
			}
		}
	})
	return
}

func (o *Operator) runstage(ctx context.Context, s cueball.State, w cueball.Worker) {
	// TODO option allowing all stages on one thread?
	log := cueball.Lc(ctx).With().Str("name", w.Name()).Logger()
	ctx = log.WithContext(ctx)
	err := w.Next(ctx)
	if err != nil && errors.Is(err, cueball.EndError) {
		w.SetStage(cueball.DONE)
		log.Debug().Interface("worker", w).Msg("Process Complete")
		o.store <- w
		return
	} else if err != nil {
		if w.Retry() {
			if true { // TODO option to bypass persistence
				w.SetStage(cueball.RETRY)
				log.Debug().Err(err).Interface("worker", w).Msg("retry persist")
				o.store <- w
				return
			}
			log.Debug().Err(err).Interface("worker", w).Msg("retry re-enqueue")
			o.intake <- w
			return
		}
		w.SetStage(cueball.FAIL)
		log.Debug().Err(err).Interface("worker", w).Msg("fail")
		o.store <- w
		return
	}
	w.SetStage(cueball.NEXT)
	log.Debug().Interface("worker", w).Msg("next stage")
	o.store <- w
	return
}
