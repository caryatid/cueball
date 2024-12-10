package worker

import (
	"context"
	"cueball"
	"github.com/google/uuid"
)

var retrymax = 3 // TODO something less lame
type Exec struct {
	Id       uuid.UUID
	Count    int
	Current  int
	Error    string // TODO internal error w/ json interfaces for persistence
	StageI    cueball.Stage
	sequence []cueball.Method
	// TODO version
}

func NewExec() *Exec {
	e := new(Exec)
	e.ID()
	return e
}

func (e *Exec) Retry() bool {
	return e.Count >= retrymax
}
func (e *Exec) ID() uuid.UUID {
	if e.Id == uuid.Nil {
		e.Id, _ = uuid.NewRandom() // TODO error handling
	}
	return e.Id
}

func (e *Exec) Next(ctx context.Context) error {
	e.Count++
	if e.Current >= len(e.sequence) {
		return new(cueball.EndError)
	}
	err := e.sequence[e.Current](ctx)
	if err != nil {
		e.Error = err.Error()
		return err
	}
	e.Error = "" // clear any previous error
	e.Current++
	if e.Current >= len(e.sequence) {
		return new(cueball.EndError)
	}
	return nil
}

func (e *Exec) Load(method ...cueball.Method) {
	e.sequence = append(e.sequence, method...)
}

func (e *Exec) Stage() cueball.Stage {
	return e.StageI
}

func (e *Exec) SetStage(st cueball.Stage) {
	e.StageI=st
}
