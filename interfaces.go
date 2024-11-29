package cueball

// NOTE: no internal imports in the file
import (
	"context"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Method func() error 

type State interface {
	Dequeue(context.Context, Worker) 
	Enqueue(context.Context, Worker) error
	Channel() chan Worker
	Persist(context.Context, Worker) error
	Group(context.Context) *errgroup.Group
}

type Executer interface {
	Next() error
	Load(method... Method)
	ID() uuid.UUID
	State(string) string
}

type Worker interface {
	Executer
	New() Worker // TODO no malloc?
	Name() string
	FuncInit() error
}


