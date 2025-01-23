package example

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/caryatid/cueball/state/log"
	"github.com/caryatid/cueball/worker"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/internal/test"
	"github.com/google/uuid"
	"testing"
	"time"
)

	

func TestWorker(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(NewCountWorker, NewStageWorker, worker.NewTestWorker)
	p, err := pipe.NewNats(ctx, test.Natsconn)
	assert.NoError(err)
	l, err := log.NewPG(ctx, test.Dbconn)
	assert.NoError(err)
	s, ctx := state.NewState(ctx, p, l, nil)
	enq := s.Start(ctx)
	var checks []uuid.UUID
	for _, wname := range cueball.Workers() {
		w := cueball.Gen(wname)
		assert.False(w.Done())
		enq <- w
		checks = append(checks, w.ID())
	}
	assert.NoError(s.Wait(ctx, time.Millisecond*150, checks))
	for _, c := range checks {
		w, _ := s.Get(ctx, c)
		assert.True(w.Done())
	}
}
