// state package for postgres persistence with NATS queue
package state

import (
	"context"
	"cueball"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/google/uuid"
)

// TODO used prepared statements
var getfmt = `SELECT worker, data FROM execution_state
WHERE id = $1
`
var persistfmt = `
INSERT INTO execution_log (id, stage, worker, data)
VALUES($1, $2, $3, $4);
`

var loadworkfmt = `
SELECT data FROM execution_state
WHERE worker = $1
AND stage = ANY($2);
`

type PG struct {
	*Op
	DB   *pgxpool.Pool
	Nats *nats.Conn
	Sub  map[string]*nats.Subscription
}

func NewPG(ctx context.Context, dburl string, natsurl string) (*PG, error) {
	var err error
	log := cueball.Lc(ctx)
	s := new(PG)
	s.Op = NewOp()
	s.Sub = make(map[string]*nats.Subscription)
	s.DB, err = pgxpool.New(ctx, dburl)
	if err != nil {
		log.Debug().Err(err).Msg("db pool error")
		return nil, err
	}
	s.Nats, err = nats.Connect(natsurl)
	if err != nil {
		log.Debug().Err(err).Msg("nats connect error")
		return nil, err
	}
	return s, nil
}

func (s *PG) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	var wb string
	var wname string
	if err := s.DB.QueryRow(ctx, getfmt, uuid).Scan(&wname, &wb); err != nil {
		return nil, err
	}
	w, ok := s.Workers()[wname]
	if !ok {
		return nil, nil
	}
	ww := w.New()
	if err := json.Unmarshal([]byte(wb), ww); err != nil {
		return nil, err
	}
	return ww, nil
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	w.ID() // TODO this sux
	b, err := json.Marshal(w)
	if err != nil {
		log.Debug().Err(err).Msg("marshal error")
		return err
	}
	// does uuid.UUID have Scanner/Valuer implemented?
	_, err = s.DB.Exec(ctx, persistfmt, w.ID().String(),
		 w.Stage().String(), w.Name(), b)
	if err != nil {
		log.Debug().Err(err).Msg("exec error")
	}
	return err
}

func (s *PG) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return s.Nats.Publish("pg."+w.Name(), data)
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
