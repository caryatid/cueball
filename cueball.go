package cueball

// NOTE: no internal imports in this package
import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"io"
)

// TODO options && config
var Worker_count = 3
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
	io.Closer
	Get(context.Context, uuid.UUID) (Worker, error)
	Persist(context.Context, Worker) error
	Enqueue(context.Context, Worker) error
	RegWorker(context.Context, Worker) error
	LoadWork(context.Context) error
}

type Operator interface {
	Work() chan Worker
	Intake() chan Worker
	Store() chan Worker
	Workers() map[string]Worker
	AddWorker(context.Context, ...Worker)
	Start(context.Context) (*errgroup.Group, context.Context)
}
