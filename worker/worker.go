package worker

import (
	"context"
	"cueball"
	"github.com/google/uuid"
	"sync"
)

type defaultExecutor struct {
	sync.Mutex
	Id       uuid.UUID
	Count    int
	Current  int
	Error    string         // TODO internal error w/ json interfaces for persistence
	StatusI  cueball.Status `json:"stage"`
	sequence []cueball.Method
	// TODO version
}

func NewExecutor() cueball.Executor {
	e := new(defaultExecutor)
	e.ID()
	return e
}

func (e *defaultExecutor) Retry() bool {
	return (e.Count - e.Current) <= cueball.RetryMax
}

func (e *defaultExecutor) ID() uuid.UUID {
	if e.Id == uuid.Nil {
		e.Id, _ = uuid.NewRandom() // TODO error handling
	}
	return e.Id
}

func (e *defaultExecutor) Next(ctx context.Context) error {
	e.Count++
	if e.Current >= len(e.sequence) {
		return cueball.EndError
	}
	err := e.sequence[e.Current](ctx)
	if err != nil {
		e.Error = err.Error()
		return err
	}
	e.Error = "" // clear any previous error
	e.Current++
	if e.Current >= len(e.sequence) {
		return cueball.EndError
	}
	return nil
}

func (e *defaultExecutor) Load(method ...cueball.Method) {
	// TODO this overwrites the sequence. Simpler than managing append, indexing, etc
	e.sequence = method
}

func (e *defaultExecutor) Status() cueball.Status {
	return e.StatusI
}

func (e *defaultExecutor) SetStatus(s cueball.Status) {
	e.StatusI = s
}
