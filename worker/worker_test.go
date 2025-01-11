package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"testing"
)

func TestWorkers(t *testing.T) {
	h, ctx := test.TSetup(t)
	cueball.RegGen(NewCountWorker, NewStageWorker)
	h.S = test.AllThree(ctx)
	for tname, s := range h.S {
		t.Run(tname, func(t *testing.T) {
			state.Start(ctx, s)
			var checks []cueball.Worker
			for _, w := range cueball.Workers() {
				if err := s.Enqueue(ctx, w); err != nil {
					h.L.Debug().Err(err).Msg("Enqueue fail")
					return
				}
				checks = append(checks, w)
			}
			h.A.NoError(state.Wait(ctx, s, checks))
		})
	}
}
