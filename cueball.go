package cueball

// NOTE: no internal imports in this package
import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// TODO options && config
var Worker_count = 4
var Lc = zerolog.Ctx // import saver; kinda dumb

type Method func(context.Context) error
type WorkFunc func(context.Context, Worker) error

type Worker interface {
	Execution
	New() Worker // TODO no malloc?
	Name() string
	FuncInit()
}

type Execution interface {
	Next(context.Context) error
	Load(...Method)
	ID() uuid.UUID
	Retry() bool
	Stage() Stage
	SetStage(Stage)
}

type State interface {
	Operator
	Get(context.Context, Worker, uuid.UUID) error
	Persist(context.Context, Worker) error
	Enqueue(context.Context, Worker) error
	Dequeue(context.Context, Worker) error
	LoadWork(context.Context, Worker, chan Worker) error
}

type Operator interface {
	Add(...Worker)
	Intake() chan Worker
	Work() chan Worker
	Workers() map[string]Worker
	Start(context.Context) *errgroup.Group
}

