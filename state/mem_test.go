package state

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/worker"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func setup(t *testing.T) (context.Context, *zerolog.Logger, *assert.Assertions, context.CancelFunc) {
	l := zerolog.New(os.Stdout)
	ctx, cancel := context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*8))
	ctx = l.WithContext(ctx)
	return ctx, cueball.Lc(ctx), assert.New(t), cancel
}

func TestState(t *testing.T) {
	ctx, log, assert, _ := setup(t)
	assert.NoError(nil)
	works := []cueball.Worker{
		new(worker.StageWorker).New(),
//		new(worker.CountWorker).New(),
//		new(worker.StageWorker).New(),
//		new(worker.CountWorker).New(),
	}
	states := map[string]cueball.State{
		"mem": func() cueball.State {
			sm, err := NewMem(ctx, works...)
			assert.NoError(err)
			return sm
		}(),
//		"pg": func() cueball.State {
//			sp, err := NewPG(ctx,
//				"postgresql://postgres:postgres@localhost:5432",
//				"nats://localhost:4222", works...)
//			assert.NoError(err)
//			return sp
//		}(),
//		"fifo": func() cueball.State {
//			sf, err := NewFifo(ctx, "fifo", ".test", works...)
//			assert.NoError(err)
//			return sf
//		}(),
	}
	for tname, s := range states {
		t.Run(tname, func(t *testing.T) {
			tick := time.NewTicker(time.Millisecond * 250)
			l := log.With().Str("test-name", tname).Logger()
			ctx = l.WithContext(ctx)
			op := NewOperator(ctx, s)
			g, ctx := op.Start(ctx)
			checks := op.Enqueue(true, works...)
			for {
				select {
				case <-ctx.Done():
					s.Close()
					return
				case <-tick.C:
					if ok := op.Done(ctx, checks...); ok {
						l.Debug().Msg("done")
						c := make(chan error)
						go func() {
							defer close(c)
							c <- g.Wait()
						}()
						select {
						case err, _ := <-c: // TODO handle ok
							if err != nil {
								l.Debug().Err(err).Msg("wait end")
							}
						case <-time.After(time.Millisecond * 750):
						}
						s.Close()
						l.Debug().Msg("all the fooking way done")
						return
					}
				}
			}
		})
	}
}
