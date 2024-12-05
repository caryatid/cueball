package cueball

// NOTE: no internal imports in the file
import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"time"
)

// TODO options && config
var worker_count = 4
var Lc = zerolog.Ctx

type Method func(context.Context) error

type Operation interface {
	Group() *errgroup.Group
	Load(Worker)
	Workers() map[string]Worker
	Channel() chan Worker
	Start(context.Context, State) error
}

type State interface {
	Operation
	// persists worker at stage
	Persist(context.Context, Worker, Stage) error
	// enqueue's a single worker
	Enqueue(context.Context, Worker) error
	// gets work from queue. if in worker set puts onto channel
	Dequeue(context.Context, Worker) error
	// gets work from persistence and puts on queue
	LoadWork(context.Context, Worker) error
}

type Execution interface {
	Next(context.Context) error
	Load(...Method)
	ID() uuid.UUID
}

type Worker interface {
	Execution
	New() Worker // TODO no malloc?
	Name() string
	FuncInit() error
}

func Start(ctx context.Context, s State) error {
	l := zerolog.New(os.Stdout) // TODO optional output
	ctx = l.WithContext(ctx)
	log := Lc(ctx) // TODO use this or just l, above?
	gl, ctxl := errgroup.WithContext(ctx)
	gd, ctxd := errgroup.WithContext(ctx)
	gr, ctxr := errgroup.WithContext(ctx)
	for _, w := range s.Workers() {
		log.Debug().Str("worker", w.Name()).Msg("load")
		gl.Go(func() error {
			tick := time.NewTicker(500 * time.Millisecond)
			for {
				select {
				case <-tick.C:
					if err := s.LoadWork(ctxl, w); err != nil {
						return err
					}
				}
			}
		})
		gd.Go(func() error {
			for {
				if err := s.Dequeue(ctxd, w); err != nil {
					return err
				}
			}
		})
	}
	for i := 0; i <= worker_count; i++ {
		gr.Go(func() error {
			for {
				select {
				case w := <-s.Channel():
					gr.Go(func() error {
						return runstage(ctxr, w, s)
					})
				}
			}
		})
	}
	return s.Group().Wait()
}

func runstage(ctx context.Context, w Worker, s State) error {
	// TODO option allowing all stages on one thread?
	err := w.Next(ctx)
	if err != nil && errors.Is(err, &EndError{}) {
		s.Persist(ctx, w, DONE)
		return nil
	} else if err != nil {
		s.Persist(ctx, w, RETRY)
		return err
	}
	s.Persist(ctx, w, NEXT)
	return nil
}
