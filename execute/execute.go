package execute

import (
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"context"
	"cueball"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"errors"
)

type Method func() error 

type EndError struct {}

func (e *EndError) Error() string {
	return "iteration complete"
}

type Exec struct {
	Id uuid.UUID
	Count int
	Current int
	Error string
	Sequence []Method `json:"-"`
}


func Run(s cueball.State) error { 
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

func (e *Exec) Group() *errgroup.Group {
	if e.Group == nil {
		e.Group, err := errgroup.WithContext(
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


