package state

import (
	"context"
	"cueball"
	"cueball/worker"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func setup(t *testing.T) (context.Context, *zerolog.Logger, *assert.Assertions, context.CancelFunc) {
	l := zerolog.New(os.Stdout)
	ctx, cancel := context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = l.WithContext(ctx)
	return ctx, cueball.Lc(ctx), assert.New(t), cancel
}

func TestMemState(t *testing.T) {
	ctx, log, assert, _ := setup(t)
	assert.NoError(nil)
	l := log.With().Str("state", "mem").Logger()
	ctx = l.WithContext(ctx)
	works := []cueball.Worker{
		new(worker.StageWorker).New(),
		new(worker.CountWorker).New(),
	}
	//sm, err := NewMem(ctx, works...)
	//if err != nil {
	//	t.Errorf("Failed %s\n", err.Error())
	//}
	sm, err := NewPG(ctx, "postgresql://postgres:postgres@localhost:5432",
		"nats://localhost:4222", works...)
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	g, ctx := sm.Start(ctx, sm)
	var checks []cueball.Worker
	for _, w := range works {
		for i := 0; i < 3; i++ {
			ww := w.New()
			ww.SetStage(cueball.INIT)
			checks = append(checks, ww)
			sm.Store() <- ww
		}
	}
	tick := time.NewTicker(time.Millisecond * 250)
	cx := true
	for {
		select {
		case <-ctx.Done():
			sm.Close()
			return
		case <-tick.C:
			cx = true
			for _, w := range checks {
				ww, _ := sm.Get(ctx, w.ID())
				if ww == nil {
					cx = false
					continue
				}
				if ww.Stage() != cueball.DONE && ww.Stage() != cueball.FAIL {
					cx = false
				}
			}
			if cx == true {
				log.Debug().Msg("done")
				sm.Close()
				//err = g.Wait()
				err := g.Wait()
				if err != nil {
					l.Debug().Err(err).Msg("wait end")
				}
				return
			}
		}
	}
}
