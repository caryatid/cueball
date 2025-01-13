// Package cueball implements an async workflow framework. The package
// permits different backing state via the [State] interface. There
// are a few provided implementations of [State].
// Users of this library will implement an object of their
// own that implements the [Worker] interface.  Generally this means
// implementing a method for each possible step of a [Worker].
// This allows, but does not mandate, complex flows
// with logic that relies on results from previous stages.
// State between workflows is managed by persisting the [Worker]
// structure's own fields.
package cueball

// NOTE: no internal imports in this package
import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"io"
	"time"
)

// TODO options && config
var (
	WorkerCount   = 3
	ChanSize      = 1
	DirectEnqueue = false
	Lc            = zerolog.Ctx // import saver; kinda dumb
)

// All methods used for stages must be of this signature
type Method func(context.Context) error

// Function that generates new works of a given type. Used to register
// workers w/ the [State] implementations
type WorkerGen func() Worker

// Worker is the interface that must be defined by clients of this library
// Worker structs will, generally, simply use the DefaultExecuter to register
// callbacks with their retry methodologies.
// TODO link examples
type Worker interface {
	Executor
	Name() string // Returns name for worker. Must be unique for a given worker group
}

// Executor provides the generalized methods to be used by all
// implementations of Worker. Most of these will be called exclusively
// by the cueball system. [Executor] is an interface so it can be embedded
// into [Worker] and implementers of worker get these methods for free.
type Executor interface {
	ID() uuid.UUID            // returns the worker's unique ID (per workload)
	Status() Status           // Gets worker status
	SetStatus(Status)         // Set's worker status
	Do(context.Context) error // Calls into the current step's retry
	GetDefer() time.Time      // calls the current steps defer
	Done() bool               // indicates, regardless of success or failure, the worker is done
}

// Retry provides an interface to allow different approaches.
// Simple counter and backoff examples are provided.
type Retry interface {
	Again() bool
	Do(context.Context) error
	Defer() time.Time
}

// State interface provides the persistence and queuing layer.
// A few implementations are provided. Most real systems will
// simply use the [state/pg] or [state/nats] implementations.
type State interface {
	WorkerSet
	ObjectStore
	io.Closer
	Start(context.Context)
	Wait(context.Context, []Worker) error
	Get(context.Context, uuid.UUID) (Worker, error) // id -> worker
	Persist(context.Context, Worker) error          // does the persistence of a worker
	Enqueue(context.Context, Worker) error          // Enqueues for processing (for work)
	LoadWork(context.Context) error                 // Scans the persistent state for workers that should be enqueued
}

type ObjectStore interface {
	Save(string, io.Reader) error
	Load(string) (io.Reader, error)
}

// WorkerSet provides the needful for generating, by name, concrete types
// with [Worker] interface definitions. Like [Executor] this is an interface
// for embedding in [State] and not intended to have multiple implementations.
type WorkerSet interface {
	Work() chan Worker
	Store() chan Worker
	Group() *errgroup.Group
}
