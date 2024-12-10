package cueball

// NOTE: no internal imports in this package
import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
//	"os"
	"time"
)

// TODO options && config
var worker_count = 4
var Lc = zerolog.Ctx // import saver; kinda dumb

type Method func(context.Context) error

type Worker interface {
	Execution
	New() Worker // TODO no malloc?
	Name() string
	FuncInit() error
}

type Execution interface {
	Next(context.Context) error
	Load(...Method)
	Stage() Stage
	SetStage(Stage)
	ID() uuid.UUID
	Retry() bool
}

type State interface {
	Operation
	// persists worker at stage
	Persist(context.Context, Worker) error
	// gets a single worker 
	Get(context.Context, uuid.UUID) (Worker, error)
	// enqueue's a single worker
	Enqueue(context.Context, Worker) error
	// gets work from queue. if in worker set puts onto channel
	Dequeue(context.Context, Worker) error
	// gets work from persistence and puts on queue
	LoadWork(context.Context, Worker) error
}

type Operation interface {
	Load(Worker)
	Workers() map[string]Worker
	Channel() chan Worker
	ReQueue(State, Worker) error
}

func Start(ctx context.Context, s State) *errgroup.Group {
	// TODO log in context or create
	// l := zerolog.New(os.Stdout) // TODO optional output
	g, ctx := errgroup.WithContext(ctx)
	log := Lc(ctx) // TODO use this or just l, above?
	for _, w := range s.Workers() {
		log.Debug().Str("worker", w.Name()).Msg("load")
		g.Go(func() error {
			tick := time.NewTicker(500 * time.Millisecond)
			for {
				select {
				case <-tick.C:
					if err := s.LoadWork(ctx, w); err != nil {
						return err
					}
				}
			}
		})
		g.Go(func() error {
			tick := time.NewTicker(500 * time.Millisecond)
			for {
				select {
				case <-tick.C: 
					if err := s.Dequeue(ctx, w); err != nil {
						return err
					}
				}
			}
		})
	}
	for i := 0; i <= worker_count; i++ {
		g.Go(func() error {
			for {
				select {
				case w := <-s.Channel():
					g.Go(func() error {
						return runstage(ctx, w, s)
					})
				}
			}
		})
	}
	return g
}

func runstage(ctx context.Context, w Worker, s State) error {
	// TODO option allowing all stages on one thread?
	err := w.Next(ctx)
	if err != nil && errors.Is(err, &EndError{}) {
		return per(ctx, s, w, DONE)
	} else if err != nil {
		if w.Retry() {
			return porq(ctx, s, w)
		} else {
			return per(ctx, s, w, DONE)
		}
	} else {
		return per(ctx, s, w, NEXT)
	}
}

func porq(ctx context.Context, s State, w Worker) error {
	if true { // TODO option
		per(ctx, w, RETRY)
	} else {
		w.SetStage(RUNNING)
		return s.Enqueue(ctx, w)
	}
}

func per(ctx context.Context, s State, w Worker, stg Stage) error {
	w.SetStage(stg)
	return s.Persist(ctx, w)
}

