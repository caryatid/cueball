package log

import (
	"context"
	"encoding/json"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO used prepared statements
var getfmt = `SELECT worker, data FROM execution_state
WHERE id = $1
`

// TODO stage -> status
var persistfmt = `
INSERT INTO execution_log (id, stage, worker, data, until)
VALUES($1, $2, $3, $4, $5);
`

var loadworkfmt__ = `
WITH x AS (SELECT id, worker, data, until
FROM execution_state WHERE stage = 'ENQUEUE' AND 
(until IS NULL OR NOW() >= until))
INSERT INTO execution_log (id, stage, worker, data, until)
SELECT id, 'INFLIGHT', worker, data, until
FROM x
RETURNING worker, data
`

var loadworkfmt = `
SELECT worker, data
FROM execution_state WHERE stage = 'ENQUEUE' AND 
(until IS NULL OR NOW() >= until)
`

type pg struct {
	DB *pgxpool.Pool
}

func NewPG(ctx context.Context, dburl string) (cueball.Record, error) {
	var err error
	l := new(pg)
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
	l.DB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return l, err
}

func (l *pg) Store(ctx context.Context, w cueball.Worker) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = l.DB.Exec(ctx, persistfmt, w.ID().String(),
		w.Status(), w.Name(), b, w.GetDefer())
	return err
}

func (l *pg) Scan(ctx context.Context, ch chan<- cueball.Worker) error {
	var wname string
	var data []byte
	rows, err := l.DB.Query(ctx, loadworkfmt)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&wname, &data); err != nil {
			return err
		}
		w, err := l.dum(wname, data)
		if err != nil {
			return err
		}
		ch <- w
	}
	return nil
}

func (l *pg) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	var wname string
	var data []byte
	if err := l.DB.QueryRow(ctx, getfmt, uuid).Scan(&wname, &data); err != nil {
		return nil, err
	}
	return l.dum(wname, data)
}

func (l *pg) Close() error {
	l.DB.Close()
	return nil
}

func (l *pg) dum(wname string, data []byte) (cueball.Worker, error) { // data unmarshal
	w := cueball.GenWorker(wname)
	if err := json.Unmarshal([]byte(data), w); err != nil {
		return nil, err
	}
	return w, nil
}
