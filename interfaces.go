package cueball

// NOTE: no internal imports in the file
import (
	"errors"
	"context"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"encoding/json"
)

type Method func() error 

type Operation interface {
	Group() *errgroup.Group
	Load(...Worker)
	Workers() []Worker
	Channel() chan Worker
}

type State interface {
	Operation
	// persists worker at stage
	Persist(context.Context, Worker, Stage) error
	// enqueue's a single worker 
	Enqueue(context.Context, Worker) error
	// below MUST be implemented as long running go routines
	// gets work from queue. if in worker set puts onto channel
	Dequeue(context.Context) 
	// gets work from persistence and puts on queue (TODO put in external process)
	LoadWork(context.Context) error 
}

type Executer interface {
	Next() error
	Load(...Method)
	ID() uuid.UUID
}

type Worker interface {
	Executer
	New() Worker // TODO no malloc?
	Name() string
	FuncInit() error
}

type EndError struct {}
type EnumError struct {}

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

var StageStr = map[int]string {0: "RUNNING", 1: "RETRY", 2: "NEXT", 3: "DONE"}
var StageInt = map[string]int {"RUNNING": 0, "RETRY": 1, "NEXT": 2, "DONE": 3}

func (s *Stage)MarshalJSON() ([]byte, error) {
	ss := StageStr[int(*s)]
	return json.Marshal(ss)
	
}

func (s *Stage)UnmarshalJSON(b []byte) error {
	i, ok := StageInt[string(b)]
	if !ok {
		return &EnumError{}
	}
	*s = Stage(i)
	return json.Unmarshal(b, s)
}

func Run(ctx context.Context, s State) error { 
	for {
		select {
		case w := <- s.Channel():
			s.Group().Go(func () error { 
				// TODO option allowing all stages on one thread?
				err := w.Next()
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
	
}

