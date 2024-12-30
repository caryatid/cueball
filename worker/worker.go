package worker

import (
//	"fmt"
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"sync"
)

type step struct {
	f cueball.Method
	Error    cueball.Error
	Retries int
}

type defaultExecutor struct {
	sync.Mutex
	Id       uuid.UUID
	Step     int
	Count    int
	StatusI  cueball.Status `json:"status"`
	Sequence []step
	// TODO version
}

func NewExecutor() cueball.Executor {
	e := new(defaultExecutor)
	e.ID()
	return e
}

func (e *defaultExecutor) Retry() bool {
	s := e.Step
	if s >= len(e.Sequence) {
		s = len(e.Sequence) - 1
	}
	return e.Sequence[s].Retries <= cueball.RetryMax
}

func (e *defaultExecutor) ID() uuid.UUID {
	if e.Id == uuid.Nil {
		e.Id, _ = uuid.NewRandom() // TODO error handling
	}
	return e.Id
}

func (e *defaultExecutor) Next(ctx context.Context) error {
	e.Count++
	if e.Step >= len(e.Sequence) {
		return cueball.NewError(cueball.EndError)
	}
	err := e.Sequence[e.Step].f(ctx)
	if err != nil {
		e.Sequence[e.Step].Error = cueball.NewError(err)
		return err
	}
	e.Step++
	if e.Step >= len(e.Sequence) {
		return cueball.NewError(cueball.EndError)
	}
	return nil
}

func (e *defaultExecutor) Load(method ...cueball.Method) {
	// TODO this overwrites the sequence. Simpler than managing append, indexing, etc
	e.Sequence = nil
	for _, m := range method {
		e.Sequence = append(e.Sequence, step{f: m})
	}
}

func (e *defaultExecutor) Status() cueball.Status {
	return e.StatusI
}

func (e *defaultExecutor) SetStatus(s cueball.Status) {
	e.StatusI = s
}
