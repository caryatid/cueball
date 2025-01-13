package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestWorkers(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(NewTestWorker)
	for tname, s := range m {
		t.Run(tname, func(t *testing.T) {
			enq := s.Start(ctx)
			var checks []uuid.UUID
			for i := 0; i < 4; i++ {
				for _, wname := range cueball.Workers() {
					w := cueball.Gen(wname)
					enq <- w
					checks = append(checks, w.ID())
				}
			}
			assert.NoError(s.Wait(ctx, time.Millisecond*140, checks))
		})
	}
}
