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
var getfmt = `SELECT stage, worker, data FROM execution_state
WHERE id = $1
`
var persistfmt = `
INSERT INTO execution_log (id, stage, worker, data)
VALUES($1, $2, $3, $4);
`

var loadworkfmt = `
SELECT worker, data FROM execution_state
WHERE worker = $1
AND stage = ANY($2) FOR UPDATE SKIP LOCKED
`

type PG struct {
	DB   *pgxpool.Pool
	Nats *nats.Conn
	Sub  map[string]*nats.Subscription
}

func NewPG(ctx context.Context, dburl string, natsurl string) (*PG, error) {
	var err error
	s := new(PG)
	s.Sub = make(map[string]*nats.Subscription)
	s.DB, err = pgxpool.New(ctx, dburl)
	if err != nil {
		return nil, err
	}
	s.Nats, err = nats.Connect(natsurl)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PG) Get(ctx context.Context, w Worker, uuid uuid.UUID) error {
	return s.DB.QueryRow(ctx, getfmt, uuid).Scan(w) {
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker, st cueball.Stage) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, persistfmt, w.ID().String(),
		 w.Stage().String(), w.Name(), b)
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
	msg, err := s.Sub[name].NextMsgWithContext(ctx)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(msg.Data, w); err != nil {
		return err
	}
	msg.AckSync()
	return nil
}

func (s *PG) LoadWork(ctx context.Context, w cueball.Worker, ch chan cueball.Worker) error {
	rows, err := s.DB.Query(ctx, loadworkfmt, w.Name(),
		[]string{cueball.RETRY.String(),
			cueball.NEXT.String()})
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		ww := w.New()
		if err := rows.Scan(ww); err != nil {
			return err
		}
		ch <- ww	
	}
	return nil
}
