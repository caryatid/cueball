package state

import (
	"context"
	"cueball"
	"errors"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

var ChanSize = 1 // Configuration? Argument to NewOperator?

type Operator struct {
	sync.Mutex
	state   cueball.State
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
	work    chan cueball.Worker
	store   chan cueball.Worker
	tick    *time.Ticker
	ltick   *time.Ticker
}

func NewOperator(w ...cueball.Worker) *Operator {
	op := new(Operator)
	op.intake = make(chan cueball.Worker, ChanSize)
	op.work = make(chan cueball.Worker, ChanSize)
	op.store = make(chan cueball.Worker, ChanSize)
	op.workers = make(map[string]cueball.Worker)
	op.tick = time.NewTicker(time.Millisecond * 50)
	op.ltick = time.NewTicker(time.Millisecond * 111)
	op.Add(w...)
	return op
}

func (o *Operator) Add(w ...cueball.Worker) {
	for _, ww := range w {
		o.workers[ww.Name()] = ww
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
	g, ctx = errgroup.WithContext(ctx_)
	// l := cueball.Lc(ctx)
	o.do(ctx, g, s)
	return
}

func (o *Operator) do(ctx context.Context, g *errgroup.Group, s cueball.State) {
	log := cueball.Lc(ctx)
	g.Go(func() error {
		for {
			if err := s.Dequeue(ctx); err != nil {
				log.Debug().Err(err).Msg("deq error")
				return err
			}
		}
	})
	g.Go(func() error {
		for {
			w := <-o.intake
			w.SetStage(cueball.ENQUEUE)
			if err := s.Enqueue(ctx, w); err != nil {
				log.Debug().Err(err).Msg("Enqueue error")
				return err
			}
			o.store <- w
		}
	})
	g.Go(func() error {
		for {
			w := <-o.store
			if err := s.Persist(ctx, w); err != nil {
				log.Debug().Err(err).Msg("Persist error")
				return err
			}
		}
	})
	g.Go(func() error {
		for {
			<-o.ltick.C
			if err := s.LoadWork(ctx); err != nil {
				log.Debug().Err(err).Msg("Load work error")
				return err
			}
		}
	})
	g.Go(func() error {
		for {
			w := <-o.work
			w.FuncInit()
			o.runstage(ctx, s, w)
		}
	})
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
