// state package for postgres persistence with NATS queue
// TODO calling uuid new too often
package state

import (
	"context"
	"cueball"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

// TODO used prepared statements
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
	DB   *pgxpool.Conn
	Nats *nats.Conn
	Sub  map[string]*nats.Subscription
}

func NewPG(ctx context.Context, g *errgroup.Group, dburl string, natsurl string) (*PG, error) {
	s := new(PG)
	s.Op = NewOp(g)
	pool, err := pgxpool.New(ctx, dburl)
	s.DB, err = pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	s.Nats, err = nats.Connect(natsurl)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker, stage cueball.Stage) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	// does uuid.UUID have Scanner/Valuer implemented?
	_, err = s.DB.Exec(ctx, persistfmt, w.ID().String(),
		w.Name(), cueball.StageStr[int(stage)], b)
	return err
}

func (s *PG) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	s.Persist(ctx, w, cueball.RUNNING)
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
		if err := s.Enqueue(ctx, ww); err != nil {
			return err
		}
	}
	return nil
}
