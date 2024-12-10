package state

import (
	"cueball"
	"sync"
	"golang.org/x/sync/errgroup"
)

var ChanSize = 1 // Configuration? Argument to NewOp?

type op struct {
	sync.Mutex
	state cueball.State
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
}

func NewOp(s cueball.State) *op {
	op := new(op)
	op.state = s
	op.intake = make(chan cueball.Worker, ChanSize)
	op.workers = make(map[string]cueball.Worker)
	return
}

func (o *op) dequeue(ctx context.Context, w cueball.Worker) error {
	ww := w.New()
	if err := o.state.Dequeue(ctx, ww); err != nil {
		return err
	}
	ww.FuncInit()
	o.intake <- ww
	return o.state.Persist(ctx, w, cueball.RUNNING)
}

func (o *op) enqueue(ctx context.Context, w cueball.Worker) error {
	if err := o.state.Enqueue(ctx, w); err != nil {
		return err
	}
	return o.state.Persist(ctx, w, cueball.ENQUEUE)
}

func (o *op) loadWork(ctx context.Context, w cueball.Worker) error {
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

func (o *op) Load(w cueball.Worker) {
	o.workers[w.Name()] = w
}

func (o *op) Channel() chan cueball.Worker {
	return o.intake
}

func (o *op) Workers() map[string]cueball.Worker {
	return o.workers
}

func (o *op) Start(ctx context.Context) *errgroup.Group {
	// TODO log in context or create
	// l := zerolog.New(os.Stdout) // TODO optional output
	g, ctx := errgroup.WithContext(ctx)
	log := Lc(ctx) // TODO use this or just l, above?
	for _, w := range o.Workers() {
		log.Debug().Str("worker", w.Name()).Msg("load")
		g.Go(func() error {
			return driver(ctx, w, o.loadWork)
		})
		g.Go(func() error {
			return driver(ctx, w, o.dequeue)
		})
	}
	for i := 0; i <= worker_count; i++ {
		g.Go(func() error {
			for {
				select {
				case <- ctx.Done():
					return nil // TODO
				case w := <-o.Channel():
					g.Go(func() error {
						return o.runstage(ctx, w)
					})
				}
			}
		})
	}
	return g
}

func (o *op)runstage(ctx context.Context, w Worker) error {
	// TODO option allowing all stages on one thread?
	err := w.Next(ctx)
	if err != nil && errors.Is(err, &EndError{}) {
		return o.state.Persist(ctx, w, cueball.DONE) // TODO FAIL state?
	} else if err != nil {
		if w.Retry() {
			if true { // TODO option to bypass persistence
				return o.state.Persist(ctx, w, cueball.RETRY)
			} 
			return o.enqueue(ctx, w)
		} 
		return o.state.Persist(ctx, w, cueball.DONE)
	} 
	return per(ctx, w, cueball.NEXT)
}

func driver(ctx context.Context, w cueball.Worker, f WorkFunc) error {
	tick := time.NewTicker(500 * time.Millisecond)
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

