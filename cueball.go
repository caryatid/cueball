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
type WorkFunc func(context.Context, Worker) error

type Worker interface {
	Execution
	New() Worker // TODO no malloc?
	Name() string
	FuncInit() error
}

type Execution interface {
	Next(context.Context) error
	Load(...Method)
	ID() uuid.UUID
	Retry() bool
}

type State interface {
	Get(context.Context, Worker, uuid.UUID) error
	Persist(context.Context, Worker, cueball.Stage) error
	Enqueue(context.Context, Worker) error
	Dequeue(context.Context, Worker) error
	LoadWork(context.Context, Worker) error
}

type Operation interface {
	Load(Worker)
	Workers() map[string]Worker
	Channel() chan Worker
}

