package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
	"strings"
	"testing"
)

var (
	Logs  = make(map[string]func(context.Context) cueball.Log)
	Pipes = make(map[string]func(context.Context) cueball.Pipe)
	Blobs = make(map[string]func(context.Context) cueball.Blob)
)

func TestStateComponents(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegWorker(ctx, worker.NewTestWorker)
	for pn, pg := range Pipes {
		for ln, lg := range Logs {
			for bn, bg := range Blobs {
				tname := strings.Join([]string{pn, ln, bn}, "-")
				s, ctx := state.NewState(ctx, pg(ctx),
					lg(ctx), bg(ctx))
				t.Run(tname+"-log", func(t *testing.T) {
					assert.NoError(TLog(ctx, s))
				})
				s, ctx = state.NewState(ctx, pg(ctx),
					lg(ctx), bg(ctx))
				t.Run(tname+"-pipe", func(t *testing.T) {
					assert.NoError(TPipe(ctx, s))
				})
			}
		}
	}
}

func TLog(ctx context.Context, s cueball.State) error {
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

func TPipe(ctx context.Context, s cueball.State) error {
	ctx, _ = context.WithCancel(ctx)
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	checks := test.Wload(enq)
	m := make(map[string]bool)
	for w := range deq {
		w.Do(ctx, s)
		if !w.Done() {
			enq <- w
		} else {
			m[w.ID().String()] = true
			cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
		}
		if len(m) >= len(checks) {
			break
		}
	}
	return s.Close()
}
