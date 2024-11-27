package cueball

// NOTE: no internal imports in the file
import (
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Method func() error 

type State interface {
	Dequeue(Worker) 
	Enqueue(Worker) error
	Channel() chan Worker
	Persist(Worker) error
	LoadState(Worker) error
}

type Executer interface {
	Next() error
	Load(method... Method)
	ID() uuid.UUID
	Group() *errgroup.Group
}

type Worker interface {
	Executer
	Name() string
	FuncInit() error
	New() Worker // TODO no malloc?
}


