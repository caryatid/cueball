package state

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"context"
	"github.com/rs/zerolog"
	"cueball"
	"cueball/worker"
	"os"
	"time"
)

func setup(t *testing.T) (context.Context, *zerolog.Logger, *assert.Assertions) {
	l := zerolog.New(os.Stdout)
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
	ctx = l.WithContext(ctx)
	return ctx, cueball.Lc(ctx), assert.New(t)
}

func TestMemState(t *testing.T) {
	ctx, log, assert := setup(t)
	assert.NoError(nil)
	l := log.With().Str("state", "mem").Logger()
	ctx = l.WithContext(ctx)
	sm, err := NewMem(ctx)
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	works := []cueball.Worker {
		new(worker.StageWorker).New(),
		new(worker.CountWorker).New(),
	}
	o := NewOperator(sm, works...)
	var checks []cueball.Worker
	for _, w := range works {
		for i:=0;i<3;i++ {
			ww := w.New()
			checks = append(checks, ww)
			sm.Persist(ctx, ww)
		}
	}
	o.Start(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("CONTEXT CANCELLED")
		default:
			for _, w := range checks {
				ww, _ := sm.Get(ctx, ww, w.ID())
			}
		}
	}
}
