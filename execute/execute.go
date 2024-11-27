package execute

import (
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"context"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"errors"
)

type Method func() error 

type EndError struct {}

func (e *EndError) Error() string {
	return "iteration complete"
}

type Executer interface {
	Next() error
	Load(method... Method)
	// TODO think on removing the below
	ID() uuid.UUID
}

type Exec struct {
	Id uuid.UUID
	Count int
	Current int
	Error string
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

func Run(s State) error { 
	g, _ := errgroup.WithContext(context.Background()) // TODO
	for {
		select {
		case w := <- s.Channel():
			g.Go(func () error { 
				err := w.Next()
				if err != nil && errors.Is(err, &EndError{}) {
					return nil
				} else if err != nil {
					log.Debug().Err(err).Interface("worker", w).
					Msg("re-enqueue")
				}
				s.Enqueue(w)
				return err
			})
		}
	}
	
}

func (e *Exec) ID() uuid.UUID {
	if e.Id == uuid.Nil {
		e.Id, _ = uuid.NewRandom() // TODO error handling
	}
	return e.Id
}

func (e *Exec) RegError(err error)  {
	e.Error = err.Error()
}

func (e *Exec) Next() error {
	e.Count++
	if e.Current >= len(e.Sequence) {
		return new(EndError)
	}
	err := e.Sequence[e.Current]()
	if err != nil {
		e.RegError(err)
		return err
	}
	e.Current++
	return nil
}

func (e *Exec) Load(method... Method) {
	e.Sequence = append(e.Sequence, method...)
}


