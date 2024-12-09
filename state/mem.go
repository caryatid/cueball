// state package using internal memory
package state

import (
	"context"
	"cueball"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/google/uuid"
)

var queue_size = 10 // TODO

type wstage {
	stage cueball.Stage
	w cueball.Worker
}

type Mem struct {
	*Op
	queue chan cueball.Worker
	ids map[string]wstage
}

func NewMem(ctx context.Context) (*Mem, error) {
	s := new(Mem)
	s.Op = NewOp()
	s.queue = make(chan Worker, queue_size)
	s.ids = make(map[string]wstage)
	return s, nil
}

func (s *Mem) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	w, ok := s.ids[uuid.String()]
	if !ok {
		return nil, nil // TODO
	}
	return w, nil
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	s.ids[w.ID().String()] = w
	return err
}

func (s *PG) Enqueue(ctx context.Context, w cueball.Worker) error {
	// functionally just a persist for mem based
	return s.Persist(ctx, w)
}

func (s *PG) Dequeue(ctx context.Context, w cueball.Worker) error {
	var err error
	name := w.Name()
	if _, ok := s.Sub[name]; !ok {
		s.Sub[name], err = s.Nats.QueueSubscribeSync("pg."+name, name)
		if err != nil {
			return err
		}
	}
	ww := w.New()
	msg, err := s.Sub[name].NextMsgWithContext(ctx)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(msg.Data, ww); err != nil {
		return err
	}
	ww.FuncInit()
	msg.AckSync()
	s.Channel() <- ww
	return nil
}

func (s *PG) LoadWork(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	rows, err := s.DB.Query(ctx, loadworkfmt, w.Name(),
		[]string{cueball.RETRY.String(),
			cueball.NEXT.String()})
	if err != nil {
		log.Debug().Err(err).Msg("query error")
		return err
	}
	defer rows.Close()
	for rows.Next() {
		ww := w.New()
		if err := rows.Scan(ww); err != nil {
			log.Debug().Err(err).Msg("scan error")
			return err
		}
		if err := s.Enqueue(ctx, ww); err != nil {
			log.Debug().Err(err).Msg("enqueue error")
			return err
		}
	}
	return nil
}
