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

type harness struct {
	L *zerolog.Logger
	A *assert.Assertions
	C context.CancelFunc
	G *errgroup.Group
}

func TSetup(t *testing.T) (h *harness, ctx context.Context) {
	l := zerolog.New(os.Stdout)
	h = new(harness)
	ctx, h.C = context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	h.G, ctx = errgroup.WithContext(ctx)
	ctx = l.WithContext(ctx)
	h.L = zerolog.Ctx(ctx)
	h.A = assert.New(t)
	return h, ctx
}

func (h *harness) TestPipe(ctx context.Context, s cueball.State) error {
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	cueball.RegGen(NewTestWorker)
	done := make(chan struct{})
	h.G.Go(func() error {
		defer close(done)
		for i := 0; i < 10; i++ {
			for _, wn := range cueball.Workers() {
				w := cueball.Gen(wn)
				enq <- w
			}
		}
		return nil

	})
	h.G.Go(func() error {
		for w := range deq {
			for i := 0; i < 2; i++ {
				err := w.Do(ctx, s)
				if err != nil {
					return err
				}
				cueball.Lc(ctx).Debug().Interface("lame", w).Send()
			}
		}
		return nil
	})
	<-done
	h.C()
	return h.G.Wait()
}
