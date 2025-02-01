package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/example"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
	"strings"
	"testing"
	"time"
)

var (
	Records = make(map[string]func(context.Context) cueball.Record)
	Pipes   = make(map[string]func(context.Context) cueball.Pipe)
	Blobs   = make(map[string]func(context.Context) cueball.Blob)
)

func TestStateComponents(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegWorker(worker.NewTestWorker, example.NewStageWorker)
	for pn, pg := range Pipes {
		for ln, lg := range Records {
			for bn, bg := range Blobs {
				tname := strings.Join([]string{pn, ln, bn}, "-")
				s, ctx := state.NewState(ctx, pg(ctx),
					lg(ctx), bg(ctx))
				t.Run(tname, func(t *testing.T) {
					assert.NoError(logT(ctx, s))
				})
				s.Close()
				/*
					s, ctx = state.NewState(ctx, nil,
						nil, bg(ctx))
					t.Run(tname+"/blob", func(t *testing.T) {
						assert.NoError(blobT(ctx, s))
					})
				*/
			}
		}
	}
}

func blobT(ctx context.Context, s cueball.State) error {
	return s.Close()
}

func logT(ctx context.Context, s cueball.State) error {
	checks := test.Wload(s.Rec())
	checks = append(checks, test.Wload(s.Enq())...)
	err := s.Wait(ctx, time.Millisecond*50, checks)
	for _, id := range checks {
		w, _ := s.Get(ctx, id)
		cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
	}
	return err
}
