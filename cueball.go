package cueball

// NOTE: no internal imports in the file
import (
	"context"
	"github.com/rs/zerolog"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"errors"
	"os"
)

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
	Dequeue(context.Context) error
	// gets work from persistence and puts on queue
	LoadWork(context.Context) error
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
	s.Group().Go(func () error {
		return s.LoadWork(ctx)
	})
	s.Group().Go(func () error {
		return s.Dequeue(ctx)
	})
	for {
		select {
		case w := <-s.Channel():
			s.Group().Go(func() error {
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
			})
		}
	}
	return s.Group().Wait()
}

