package execute

import (
	// "github.com/rs/zerolog/log"
	"context"
	"cueball"
	"github.com/google/uuid"
	"errors"
)


type Exec struct {
	Id uuid.UUID
	Count int
	Current int
	Error string
	state cueball.States
	Sequence []cueball.Method `json:"-"`
}


func Run(s cueball.State) error { 
	ctx := context.Background()
	for {
		select {
		case w := <- s.Channel():
			s.Group(ctx).Go(func () error { 
				err := w.Next()
				if err != nil && errors.Is(err, &cueball.EndError{}) {
					return nil
				} else if err != nil {
					w.SetState("RETRY")
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

func (e *Exec) SetState (s string) error {
	i, ok := cueball.StateInt[s]
	if !ok { 
		return new(cueball.EnumError)
	}
	e.state	= cueball.States(i)
	return nil
}

func (e *Exec)State () string {
	return cueball.StateStr[int(e.state)] 
}

func (e *Exec) RegError(err error)  {
	e.Error = err.Error()
}

func (e *Exec) Next() error {
	e.Count++
	if e.Current >= len(e.Sequence) {
		return new(cueball.EndError)
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


