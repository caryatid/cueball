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
	Records = make(map[string]func(context.Context) cueball.Record)
	Pipes   = make(map[string]func(context.Context) cueball.Pipe)
	Blobs   = make(map[string]func(context.Context) cueball.Blob)
)

func TestStateComponents(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegWorker(worker.NewTestWorker)
	for pn, pg := range Pipes {
		for ln, lg := range Records {
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
	checks := test.Wload(s.Rec())
	for !s.Check(ctx, checks) {
		for w := range s.RunScan(ctx) {
			w.Do(ctx, s)
			s.Rec() <- w
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
	m := make(map[string]bool)
	test.Wload(s.Enq())
	for w := range s.Deq() {
		cueball.Lc(ctx).Debug().Interface(" x ", w).Send()
		w.Do(ctx, s)
		cueball.Lc(ctx).Debug().Interface(" z ", w).Send()
		if !w.Done() {
			cueball.Lc(ctx).Debug().Interface(" y ", w).Send()
			s.Enq() <- w
		} else {
			m[w.ID().String()] = true
			cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
		}
		if len(m) >= len(cueball.Workers()) {
			break
		}
	}
	return s.Close()
}
