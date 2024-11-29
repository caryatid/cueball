package execute

import (
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"context"
	"cueball"
	"github.com/google/uuid"
	"errors"
)


type EndError struct {}

func (e *EndError) Error() string {
	return "iteration complete"
}

type StateStr string

type Exec struct {
	Id uuid.UUID
	Count int
	Current int
	Error string
	state StateStr
	Sequence []cueball.Method `json:"-"`
}


func Run(s cueball.State) error { 
	ctx := context.Background()
	for {
		select {
		case w := <- s.Channel():
			w.State("START")
			s.Group(ctx).Go(func () error { 
				err := w.Next()
				if err != nil && errors.Is(err, &EndError{}) {
					w.State("DONE")
					return nil
				} else if err != nil {
					w.State("RETRY")
					log.Debug().Err(err).Interface("worker", w).
					Msg("re-enqueue")
				}
				s.Persist(ctx, w)
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

func (e *Exec) State (s string) string {
	if s != "" {
		e.state = StateStr(s) 
	}
	return string(e.state)
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

func (e *Exec) Load(method... cueball.Method) {
	e.Sequence = append(e.Sequence, method...)
}


