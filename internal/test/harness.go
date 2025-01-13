package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"os"
	"testing"
	"time"
)

var Dbconn = "postgresql://postgres:postgres@localhost:5432"
var Natsconn = "nats://localhost:4222"

func TSetup(t *testing.T) (*assert.Assertions, context.Context) {
	ctx, _ := context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = zerolog.New(os.Stdout).WithContext(ctx)
	return assert.New(t), ctx
}

func Pipe(ctx context.Context, s cueball.State,
	ws ...cueball.Worker) error {
	ctx, c := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	g.Go(func() error {
		defer func() {
			time.Sleep(time.Millisecond * 100)
			s.Close()
			c()
		}()
		for _, w := range ws {
			enq <- w
		}
		close(enq)
		return nil
	})
	g.Go(func() error {
		for w := range deq {
			if err := doN(ctx, s, w, 2); err != nil {
				return err
			}
			cueball.Lc(ctx).Debug().
				Interface("worker", w).Msg("pipe-test")
		}
		return nil
	})
	return g.Wait()
}

func Log(ctx context.Context, s cueball.State,
	ws ...cueball.Worker) error {
	store := s.Run(ctx, s.Store)
	scan := s.Run(ctx, s.Scan)
	for _, w := range ws {
		w.SetStatus(cueball.ENQUEUE)
		store <- w
	}
	for w := range scan {
		doN(ctx, s, w, 2)
		store <- w
		cueball.Lc(ctx).Debug().
			Interface("worker", w).Msg("log-scan")
	}
	for _, w := range ws {
		ww, _ := s.Get(ctx, w.ID())
		cueball.Lc(ctx).Debug().
			Interface("worker", ww).Msg("log-get")
	}
	return nil
}

func doN(ctx context.Context, s cueball.State, w cueball.Worker, cnt int) error {
	for i := 0; i < cnt; i++ {
		if err := w.Do(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
