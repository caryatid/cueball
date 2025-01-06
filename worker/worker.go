package worker

import (
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
)

type defaultExecutor struct {
	cueball.Step
	Id      uuid.UUID
	Count   int
	StatusI cueball.Status `json:"status"`
	// TODO version
}

func NewExecutor(name string) cueball.Executor {
	e := new(defaultExecutor)
	e.ID()
	e.Step = new(step)
	return e
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
