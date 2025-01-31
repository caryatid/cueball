package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
	"strings"
	"testing"
	"time"
)

var (
	Logs  = make(map[string]func(context.Context) cueball.Log)
	Pipes = make(map[string]func(context.Context) cueball.Pipe)
	Blobs = make(map[string]func(context.Context) cueball.Blob)
)

func TestStateComponents(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegWorker(worker.NewTestWorker)
	for pn, pg := range Pipes {
		for ln, lg := range Logs {
			for bn, bg := range Blobs {
				tname := strings.Join([]string{pn, ln, bn}, "-")
				s, ctx := state.NewState(ctx, nil,
					lg(ctx), nil)
				t.Run(tname+"/log", func(t *testing.T) {
					assert.NoError(logT(ctx, s))
				})
				s, ctx = state.NewState(ctx, pg(ctx),
					nil, nil)
				t.Run(tname+"/pipe", func(t *testing.T) {
					assert.NoError(pipeT(ctx, s))
				})
				s, ctx = state.NewState(ctx, nil,
					nil, bg(ctx))
				t.Run(tname+"/blob", func(t *testing.T) {
					assert.NoError(blobT(ctx, s))
				})
			}
		}
	}
}

func blobT(ctx context.Context, s cueball.State) error {
	return s.Close()
}

func logT(ctx context.Context, s cueball.State) error {
	store := s.Run(ctx, s.Store)
	checks := test.Wload(store)
	for !s.Check(ctx, checks) {
		for w := range s.Run(ctx, s.Scan) {
			w.Do(ctx, s)
			store <- w
		}
	}
	for _, id := range checks {
		w, _ := s.Get(ctx, id)
		cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
	}
	return s.Close()
}

func pipeT(ctx context.Context, s cueball.State) error {
	ctx, _ = context.WithCancel(ctx)
	deq := s.Run(ctx, s.Dequeue)
	enq := s.Run(ctx, s.Enqueue)
	m := make(map[string]bool)
	c := make(chan error)
	go func() {
		for w := range deq {
			cueball.Lc(ctx).Debug().Interface(" x ", w).Send()
			w.Do(ctx, s)
			if !w.Done() {
				cueball.Lc(ctx).Debug().Interface(" y ", w).Send()
				enq <- w
			} else {
				m[w.ID().String()] = true
				cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
			}
			if len(m) >= len(cueball.Workers()) {
				break
			}
		}
		c <- nil
	}()
	time.Sleep(time.Millisecond * 50)
	test.Wload(enq)
	<-c
	return s.Close()
}
