package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/caryatid/cueball/state/log"
	"github.com/caryatid/cueball/internal/test"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestWorkers(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(NewTestWorker)
	p, err := pipe.NewNats(ctx, test.Natsconn)
	assert.NoError(err)
	l, err := log.NewPG(ctx, test.Dbconn)
	assert.NoError(err)
	s, ctx := state.NewState(ctx, p, l, nil)
	t.Run("top-level", func(t *testing.T) {
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
