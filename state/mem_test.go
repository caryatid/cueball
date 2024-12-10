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

func setup() (context.Context, *zerolog.Logger) {
	l := zerolog.New(os.Stdout)
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
	ctx = l.WithContext(ctx)
	log := cueball.Lc(ctx)
	return ctx, log
}

func TestMemState(t *testing.T) {
	ctx, log := setup()
	assert := assert.New(t)
	log.Debug().Str("name", t.Name()).Msg("starting")
	sm, err := NewMem(ctx)
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	w_ := new(worker.StageWorker)
	l := log.With().Str("state", "mem").Logger()
	ctx = l.WithContext(ctx)
	sm.Load(w_)
	for i:=0; i<=3; i++ {
		w := w_.New()
		w.SetStage(cueball.NEXT)
		assert.NoError(sm.Persist(ctx, w))
	}
	cueball.Start(ctx, sm)
	select {
	case <-ctx.Done():
		log.Debug().Msg("CONTEXT CANCELLED")
	}
}
