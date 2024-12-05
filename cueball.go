package cueball

// NOTE: no internal imports in the file
import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"errors"
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

type EndError struct{}
type EnumError struct{}

func (e *EndError) Error() string {
	return "iteration complete"
}

func (e *EnumError) Error() string {
	return "invalid enum value"
}

type Stage int

const (
	RUNNING Stage = iota
	RETRY
	NEXT
	DONE
)

var StageStr = map[int]string{0: "RUNNING", 1: "RETRY", 2: "NEXT", 3: "DONE"}
var StageInt = map[string]int{"RUNNING": 0, "RETRY": 1, "NEXT": 2, "DONE": 3}

func (s *Stage) MarshalJSON() ([]byte, error) {
	ss := StageStr[int(*s)]
	return json.Marshal(ss)
}

func (s Stage) String() string {
	return StageStr[int(s)]
}

func (s *Stage) UnmarshalJSON(b []byte) error {
	i, ok := StageInt[string(b)]
	if !ok {
		return &EnumError{}
	}
	*s = Stage(i)
	return json.Unmarshal(b, s)
}

func Start(ctx context.Context, s State) error {
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

