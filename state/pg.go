// state package for postgres persistence with NATS queue
// TODO calling uuid new too often
package state

import (
	//	"github.com/rs/zerolog"
	"context"
	"cueball"
	"encoding/json"
	"golang.org/x/sync/errgroup"
	// "database/sql" maybe we avoid?
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"time"
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
	DB *pgxpool.Conn
	Nats *nats.Conn
	Sub map[string]*nats.Subscription
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

func (s *PG) Dequeue(ctx context.Context) error {
	g, _ := errgroup.WithContext(ctx)
	for name, w := range s.Workers() {
		var err error
		if _, ok := s.Sub[name]; !ok {
			s.Sub[name], err = s.Nats.QueueSubscribeSync("pg."+name, name) 
			if err != nil {
				return err
			}
		}
		g.Go(func () error {
			ww := w.New()
			for {
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
			}
		})
	}
	return g.Wait()
}

func (s *PG) LoadWork(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx) 
	for name, w := range s.Workers() {
		g.Go(func () error {
			tick := time.NewTicker(500 * time.Millisecond)
			for { 
				select {
				case <-tick.C:
					err := func () error { // anonfunc to facilitate defers
						s.Lock()
						defer s.Unlock()
						// TODO begin transaction.
						rows, err := s.DB.Query(ctx, loadworkfmt, name,
							[]string{cueball.RETRY.String(),
								cueball.NEXT.String()})
						if err != nil {
							return err	
						}
						defer rows.Close()
						ww := w.New()
						for rows.Next() {
							if err := rows.Scan(ww); err != nil {
								return err
							}
							if err := s.Enqueue(ctx, ww); err != nil {
								return err
							}
						}
						return nil
					}()
					return err
				}
			}
		})
	}
	return g.Wait()
}

