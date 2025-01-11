package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"testing"
)

func TestWorkers(t *testing.T) {
	h, ctx := test.TSetup(t)
	cueball.RegGen(NewCountWorker, NewStageWorker)
	for tname, s := range test.AllThree(ctx) {
		t.Run(tname, func(t *testing.T) {
			s.Start(ctx)
			var checks []cueball.Worker
			for _, w := range cueball.Workers() {
				if err := s.Enqueue(ctx, w); err != nil {
					h.L.Debug().Err(err).Msg("Enqueue fail")
					return
				}
				checks = append(checks, w)
			}
			h.A.NoError(s.Wait(ctx, checks))
		})
	}
}
