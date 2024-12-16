// state package for postgres persistence with NATS queue
package state

import (
	"context"
	"cueball"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
	"sync"
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
SELECT worker, data FROM execution_state
WHERE stage = ANY($1); -- FOR UPDATE SKIP LOCKED
`

type PG struct {
	*Operator
	DB   *pgxpool.Pool
	Nats *nats.Conn
	sub  sync.Map
	ch   chan *nats.Msg
}

func NewPG(ctx context.Context, dburl string, natsurl string, w ...cueball.Worker) (*PG, error) {
	var err error
	s := new(PG)
	s.Operator = NewOperator(w...)
	config, err := pgxpool.ParseConfig(dburl)
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		dt, err := conn.LoadType(ctx, "stage")
		if err != nil {
			return err
		}
		conn.TypeMap().RegisterType(dt)
		dta, err := conn.LoadType(ctx, "_stage")
		if err != nil {
			return err
		}
		conn.TypeMap().RegisterType(dta)
		return nil
	}
	s.DB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	s.Nats, err = nats.Connect(natsurl)
	if err != nil {
		return nil, err
	}
	return s, err
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, persistfmt, w.ID().String(),
		w.Stage(), w.Name(), b)
	return err
}

func (s *PG) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return s.Nats.Publish("cueball.pg."+w.Name(), data)
}

func (s *PG) newsub(name string) error {
	ss := "cueball.pg." + name
	sub, err := s.Nats.QueueSubscribeSync(ss, ss)
	if err != nil {
		return err
	}
	s.sub.Store(name, sub)
	return nil
}

func (s *PG) Dequeue(ctx context.Context) error {
	// l := cueball.Lc(ctx)
	g, ctx := errgroup.WithContext(ctx)
	for name, _ := range s.Workers() {
		if _, ok := s.sub.Load(name); !ok {
			s.newsub(name)
		}
		sub_, _ := s.sub.Load(name)
		sub := sub_.(*nats.Subscription)
		if !sub.IsValid() {
			sub.Unsubscribe()
			sub.Drain()
			s.newsub(name)
		}
	}
	s.sub.Range(func(name_, sub_ any) bool {
		sub := sub_.(*nats.Subscription)
		name := name_.(string)
		g.Go(func() error {
			for {
				msg, err := sub.NextMsgWithContext(ctx)
				if err != nil {
					return err
				}
				w := s.Workers()[name].New()
				if err := json.Unmarshal(msg.Data, w); err != nil {
					return err
				}
				s.Work() <- w
			}
		})
		return true
	})
	return g.Wait()
}

func (s *PG) Close() error {
	s.sub.Range(func(_, sub_ any) bool {
		sub := sub_.(*nats.Subscription)
		sub.Unsubscribe()
		sub.Drain()
		return true
	})
	s.Nats.Close()
	s.DB.Close()
	return nil
}

func (s *PG) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	var wname string
	var data string
	if err := s.DB.QueryRow(ctx, getfmt, uuid).Scan(&wname, &data); err != nil {
		return nil, err
	}
	return s.dum(wname, data)
}

func (s *PG) LoadWork(ctx context.Context) error {
	var wname string
	var data string
	rows, err := s.DB.Query(ctx, loadworkfmt,
		[]cueball.Stage{cueball.RETRY, cueball.NEXT, cueball.INIT})
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&wname, &data); err != nil {
			return err
		}
		w, err := s.dum(wname, data)
		if err != nil {
			return err
		}
		s.Intake() <- w
	}
	return nil
}

func (s *PG) dum(wname, data string) (cueball.Worker, error) { // data unmarshal
	w := s.Workers()[wname].New()
	if err := json.Unmarshal([]byte(data), w); err != nil {
		return nil, err
	}
	return w, nil
}
