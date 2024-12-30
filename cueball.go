// Package cueball implements an async workflow framework. The package
// permits different backing state via the [State] interface. There
// are a few provided implementations of [State].
// Users of this library will generally implement an object of their
// own that implements the [Worker] interface.  Generally this means
// implementing a method for each possible step of a [Worker]. Those
// methods are of type [Method] and registered at each stage by
// [Worker.StageInit()]. This allows, but does not mandate, complex flows
// with logic that relies on results from previous stages.
// State between workflows is managed by persisting the [Worker] structure's
// fields. 
package cueball

// NOTE: no internal imports in this package
import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"io"
)

// TODO options && config
var (
	RetryMax    = 3
	WorkerCount = 3
	ChanSize    = 1
	DirectEnqueue = false
	Lc          = zerolog.Ctx // import saver; kinda dumb
)

// All methods used for stages must be of this signature
type Method func(context.Context) error

// Worker is the interface that must be defined by clients of this library
// Worker structs will, generally, add the DefaultExecuter
type Worker interface {
	Executor
	New() Worker  // Must create a new empty worker.
	Name() string // Returns name for worker. Must be unique for a given worker group
	StageInit()   // Loads up the work set. Used to specify next possible work for a stage
}

// Executor provides the generalized methods to be used by all
// implementations of Worker. Most of these will be called exclusively
// by the cueball system.
type Executor interface {
	Load(Step)             // Sets method set for the worker
	ID() uuid.UUID              // returns the worker's unique ID (per workload)
	Status() Status             // Gets worker status
	SetStatus(Status)           // Set's worker status
	Step() Step
}

type Step interface {
	Do(context.Context) error
	Current() Step
	SetNext(Step)
	Tries() int
	Done() bool
	Err() error 
}

// State interface provides the persistence and queuing layer.
type State interface {
	io.Closer
	WorkerSet
	Get(context.Context, uuid.UUID) (Worker, error) // returns a worker given an ID
	Persist(context.Context, Worker) error          // does the persistence of a worker
	Enqueue(context.Context, Worker) error          // Enqueues for processing (for work)
	LoadWork(context.Context, chan Worker) error    // Scans the persistent state for workers that should be enqueued
}

// WorkerSet represents the generalized methods to be used
// by implementations of State. State structs will, generally,
// add the DefaultWorkerSet as an embedded type
type WorkerSet interface {
	AddWorker(context.Context, ...Worker) error
	ByName(name string) Worker
	List() []Worker
	Out() chan Worker
}
