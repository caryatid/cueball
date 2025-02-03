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
					assert.NoError(compT(ctx, s))
				})
			}
		}
	}
}

func compT(ctx context.Context, s cueball.State) error {
	checks := test.Wload(s.Enq())
	if err := s.Wait(ctx, time.Millisecond*25, checks); err != nil {
		return err
	}

	for _, id := range checks {
		w, _ := s.Get(ctx, id)
		cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
	}
	return s.Close()
}
