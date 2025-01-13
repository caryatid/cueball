package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestWorkers(t *testing.T) {
	h, ctx := test.TSetup(t)
	cueball.RegGen(NewCountWorker, NewStageWorker)
	m := test.AllThree(ctx, t)
	for tname, s := range m {
		t.Run(tname, func(t *testing.T) {
			enq := s.Start(ctx)
			var checks []uuid.UUID
			for _, wname := range cueball.Workers() {
				w := cueball.Gen(wname)
				enq <- w
				checks = append(checks, w.ID())
			}
			h.A.NoError(s.Wait(ctx, time.Millisecond * 50, checks))
		})
	}
}
