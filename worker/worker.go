package worker

import (
//	"fmt"
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"sync"
)

type basicStep {
	f cueball.Method
	Error    cueball.Error
	Tries int
	Complete bool
	next cueball.Step
}

func (s *basicStep) Tries () int {
	return s.Tries
}

func (s *basicStep) Current() cueball.Step {
	if s.next == nil || s.Complete == false {
		return s
	}
	return s.next.Current()
}

func (s *basicStep) Do(ctx context.Context) error {
	if s.Complete { // TODO how to handle per step done-ness
		return nil 
	}
	s.Tries++
	err := cs.f(ctx)
	if err != nil {
		err = s.SetError(cueball.NewError(err))
	} else {
		s.Complete = true
	}
	return err
}

func (s *basicStep)Done() bool {
	ss := s.Current()
	return s.Complete && s.next == nil
}

func (s *basicStep)SetNext(cs cueball.Step) cueball.Step {
	s.next = cs
	return s.next
}

func BasicStep(m cueball.Method) cueball.Step {
	return &basicStep{f: m}
}

type defaultExecutor struct {
	sync.Mutex
	Id       uuid.UUID
	Count    int
	StatusI  cueball.Status `json:"status"`
	flow cueball.Step
	// TODO version
}

func NewExecutor() cueball.Executor {
	e := new(defaultExecutor)
	e.ID()
	return e
}

func (e *defaultExecutor) Next(ctx context.Context) error {
	err := e.flow.Current().Do()
	if err 
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

func (e *defaultExecutor) Status() cueball.Status {
	return e.StatusI
}

func (e *defaultExecutor) SetStatus(s cueball.Status) {
	e.StatusI = s
}
