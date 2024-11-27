package state

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"context"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
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

func TestState(t *testing.T) {
	ctx, log := setup()
	assert := assert.New(t)
	log.Debug().Str("name", t.Name()).Msg("starting")
	sf, err := NewFifo(ctx, "test-fifo", ".test")
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	sp, err := NewPG(ctx, "postgresql://postgres:postgres@localhost:5432",
		"nats://localhost:4222")
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	tests := map[string]*struct {
		s cueball.State
		g *errgroup.Group
	} {
		"fifo": { s: sf },
		"pg": { s: sp },
	}
	w_ := new(worker.StageWorker)
	for name, s := range tests {
		l := log.With().Str("state", name).Logger()
		ctx := l.WithContext(ctx)
		t.Run(name, func (t *testing.T) {
			s.s.Load(w_)
			for i:=0; i<=3; i++ {
				w := w_.New()
				assert.NoError(s.s.Persist(ctx, w, cueball.NEXT))
			}
			cueball.Start(ctx, s.s)
		})
	}
	select {
	case <-ctx.Done():
		log.Debug().Msg("CONTEXT CANCELLED")
	}
}
