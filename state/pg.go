// state package for postgres persistence with NATS queue
// TODO calling uuid new too often
package state

import (
	//	"github.com/rs/zerolog"
	"bufio"
	"context"
	"cueball"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	// "database/sql" maybe we avoid?
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"time"
)

type PG struct {
	*Op
	DB *pgx.Conn
	Nats *nats.Conn
}

func NewPG(ctx context.Context, g *errgroup.Group, dburl string, natsurl string) (*PG, error) {
	var err error
	s := new(PG)
	s.Op = NewOp(g)
	s.DB, err = pgxpool.New(ctx, dburl) 
	if err != nil {
		return nil, err
	}
	s.Nats, err = nats.Connection(natsurl)
	if err != nil {
		return nil, err
	}
	return nil
}

func (s *PG) Persist(ctx context.Context, w cueball.Worker, stage cueball.Stage) error {
	persistfmt = `
		INSERT INTO execution_log (id, stage, worker)
		VALUES($1, $2, $3);
	`
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	// does uuid.UUID have Scanner/Valuer implemented?
	_, err := s.DB.Exec(ctx, persistfmt, w.ID().String(), cueball.StageStr[int(stage)], b)
	return err
}

func (s *PG) Dequeue(ctx context.Context) error {
	s.Nats.Subscribe(
	data, err := bufio.NewReader(s.out).ReadString('\n')
	if err != nil {
		log.Debug().Err(err).Msg("failed reading")
		return err
	}
	p := new(Pack)
	if err := unmarshal(data, p); err != nil {
		return err
	}
	if w, ok := s.Workers()[p.Name]; ok {
		ww := w.New()
		if err := unmarshal(p.Codec, ww); err != nil {
			return err
		}
		ww.FuncInit()
		s.Channel() <- ww
	} else { // re-enqueue if not a known worker type. stop infinite loops, TTL?
		if err := s.enqueue(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *Fifo) enqueue(p Pack) error { // allows re-queuing a packed item
	data, err := marshsal(p)
	if _, err = s.in.Write(append(data, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return s.in.Sync()
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.Lock()
	defer s.Unlock()
	data, err := marshal(w)
	return s.enqueue(&Pack{Name: w.Name(), Codec: data})
}

func (s *Fifo) LoadWork(ctx context.Context) {
	for name, w := range s.Workers() {
		dir := pre + "/" + w.Name()
		if err := os.Mkdir(dir, 0700); err != nil {
			return err
		}
		files, _ := os.ReadDir(dir)
		var curid string
		var pret int
		x := make(map[string]string)
		for _, f := range files {
			id, ts, _ := strings.Split(f, ":")
			curt, _ := strconv.Atoi(ts)
			if id != curid {
				curid = id
				x[id]f.Name()
				continue
			} else if curt > pret {
				x[id]f.Name()
			}
		}
		for _, f := range x {
			_, _, stage := strings.Split(f, ":")
			if stage == "NEXT" || stage == "RETRY" {
				// READ, enqueue, persist as RUNNING
			}
		}
		
	}
}

