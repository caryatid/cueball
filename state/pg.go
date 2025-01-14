package state

import (
	"context"
	"encoding/json"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

// pub/sub key prefix for nats
var prefix = "cueball.pg."

// TODO used prepared statements
var getfmt = `SELECT worker, data FROM execution_state
WHERE id = $1
`

// TODO stage -> status
var persistfmt = `
INSERT INTO execution_log (id, stage, worker, data, until)
VALUES($1, $2, $3, $4, $5);
`

var loadworkfmt = `
WITH x AS (SELECT id, worker, data, until 
FROM execution_state WHERE stage = 'ENQUEUE' AND 
(until IS NULL OR NOW() >= until))
INSERT INTO execution_log (id, stage, worker, data, until)
SELECT id, 'INFLIGHT', worker, data, until
FROM x
RETURNING worker, data;
`

type pg struct {
	cueball.WorkerSet
	DB   *pgxpool.Pool
	Nats *nats.Conn
	sub  sync.Map
}

func NewPG(ctx context.Context, dburl string, natsurl string,
	works ...cueball.WorkerGen) (cueball.State, error) {
	var err error
	s := new(pg)
	s.WorkerSet = DefaultWorkerSet(works...)
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
	go s.dequeue(ctx)
	return s, err
}

func (s *pg) Persist(ctx context.Context, w cueball.Worker) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, persistfmt, w.ID().String(),
		w.Status(), w.Name(), b, w.GetDefer())
	return err
}

func (s *pg) Enqueue(ctx context.Context, w cueball.Worker) error {
	w.SetStatus(cueball.INFLIGHT)
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	return s.Nats.Publish(prefix+w.Name(), data)
}

func (s *pg) LoadWork(ctx context.Context) error {
	var wname string
	var data string
	rows, err := s.DB.Query(ctx, loadworkfmt)
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
		s.Enqueue(ctx, w)
	}
	return nil
}

func (s *pg) checksubs(ctx context.Context, g *errgroup.Group) error {
	for name, _ := range s.Workers() {
		sub_, ok := s.sub.Load(name)
		if !ok || !sub_.(*nats.Subscription).IsValid() {
			if ok {
				sub_.(*nats.Subscription).Unsubscribe()
				sub_.(*nats.Subscription).Drain()
			}
			ss := prefix + name
			sub, err := s.Nats.QueueSubscribeSync(ss, ss)
			if err != nil {
				return err
			}
			s.sub.Store(name, sub)
			g.Go(func() error {
				for {
					msg, err := sub.NextMsgWithContext(ctx)
					if err != nil {
						return err
					}
					w := s.NewWorker(name)
					if err := json.Unmarshal(msg.Data, w); err != nil {
						return err
					}
					s.Work() <- w
				}
			})
		}
	}
	return nil
}

func (s *pg) dequeue(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	t := time.NewTicker(time.Millisecond * 15)
	if err := s.checksubs(ctx, g); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return g.Wait()
		case <-t.C:
			if err := s.checksubs(ctx, g); err != nil {
				return err
			}
		}
	}
}

func (s *pg) Close() error {
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

func (s *pg) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	var wname string
	var data string
	if err := s.DB.QueryRow(ctx, getfmt, uuid).Scan(&wname, &data); err != nil {
		return nil, err
	}
	return s.dum(wname, data)
}

func (s *pg) dum(wname, data string) (cueball.Worker, error) { // data unmarshal
	w := s.NewWorker(wname)
	if err := json.Unmarshal([]byte(data), w); err != nil {
		return nil, err
	}
	return w, nil
}
