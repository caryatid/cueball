package state

import (
	"testing"
	"context"
	"golang.org/x/sync/errgroup"
	"github.com/rs/zerolog"
	"cueball"
	"cueball/worker"
	"os"
)



func TestFifoCore(t *testing.T) {
	g, ctx_ := errgroup.WithContext(context.Background())
	l := zerolog.New(os.Stdout).Level(zerolog.DebugLevel).
		With().Str("test", t.Name()).Logger()
	ctx := l.WithContext(ctx_)
	log := cueball.Lc(ctx)
	s, err := NewFifo(ctx, g, "test-fifo", ".test")
	if err != nil {
		t.Errorf("Failed %s\n", err.Error())
	}
	w_ := new(worker.StageWorker)
	w := w_.New()
	s.Load(w)
	s.Persist(ctx, w, cueball.RUNNING)
	s.Persist(ctx, w, cueball.DONE)
	ww := w.New()
	s.Persist(ctx, ww, cueball.NEXT)
	ww = w.New()
	s.Persist(ctx, ww, cueball.NEXT)
	go s.LoadWork(ctx, w)
	s.Dequeue(ctx, w)
	<-s.Channel()
	s.Dequeue(ctx, w)
	y := <-s.Channel()
	log.Debug().Interface("worker", y).Msg("end of chain")
}
