package execute

import (
	"context"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Method func() error 

type EndError struct {}

func (e *EndError) Error() string {
	return "iteration complete"
}


type Executer interface {
	ID() uuid.UUID
	GenID() error
	Next(State, Worker) error
	Load(method... Method)
	RegError(error)
}

type Exec struct {
	Id uuid.UUID
	Count int
	Current int
	Error error
	Sequence []Method `json:"-"`
}

type Worker interface {
	Executer
	Name() string
	FuncInit() error
	New() Worker // TODO no malloc?
}

type State interface {
	Dequeue(Worker) error 
	Enqueue(Worker) error
	Channel() chan Worker

	// should be able todo w/ only queues
	// Persist(Worker) error
	// LoadState(Worker) error
}

func Run(s State, w Worker) error { 
	go s.Dequeue(w)
	g, _ := errgroup.WithContext(context.Background()) // TODO
	for {
		select {
		case w = <- s.Channel():
			g.Go(func () error { return w.Next(s, w) })
		// other cases
		}
	}
	
}

func (e *Exec) ID() uuid.UUID {
	return e.Id
}

func (e *Exec) RegError(err error)  {
	e.Error = err
}

func (e *Exec) GenID() error {
	var err error
	e.Id, err = uuid.NewRandom()
	return err
}

func (e *Exec) Next(s State, w Worker) error {
	e.Count++
	if e.Current >= len(e.Sequence) { // TODO test for greater; should never happen
		return new(EndError)
	}
	err := e.Sequence[e.Current]()
	if err != nil {
		e.RegError(err)
		s.Enqueue(w)
		return err
	}
	e.Current++
	err = s.Enqueue(w)
	return nil
}

func (e *Exec) Load(method... Method) {
	e.Sequence = append(e.Sequence, method...)
}


