package cueball

// NOTE: no internal imports in the file
import (
	"context"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"encoding/json"
)

type Method func() error 

type State interface {
	Operations
	// gets work from queue. if in worker set puts onto channel
	Dequeue(context.Context) 
	// gets work from persistence and puts on queue 
	Enqueue(context.Context) error
	// persists worker at stage
	Persist(context.Context) error
}

type Operations interface {
	Group(context.Context) *errgroup.Group
	Channel() chan Worker
	Workers() []Worker
	Load(...Worker)
}

type Executer interface {
	Next() error
	Load(...Method)
	ID() uuid.UUID
	SetState(string) error
	State() string
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

type States int

const (
	START States = iota
	RETRY
	RUNNING
	DONE
)

var StateStr = map[int]string {0: "START", 1: "RETRY", 2: "RUNNING", 3: "DONE"}
var StateInt = map[string]int {"START": 0, "RETRY": 1, "RUNNING": 2, "DONE": 3}

func (s *States)MarshalJSON() ([]byte, error) {
	ss := StateStr[int(*s)]
	return json.Marshal(ss)
	
}

func (s *States)UnmarshalJSON(b []byte) error {
	i, ok := StateInt[string(b)]
	if !ok {
		return &EnumError{}
	}
	*s = States(i)
	return json.Unmarshal(b, s)
}
