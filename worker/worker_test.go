package worker

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(NewTestWorker)
	for _, wname := range cueball.Workers() {
		w := cueball.Gen(wname)
		assert.NoError(uuid.Validate(w.ID().String()))
		assert.IsType(w.Status(), cueball.ENQUEUE)
		w.SetStatus(cueball.INFLIGHT)
		assert.Equal(w.Status(), cueball.INFLIGHT)
		assert.False(w.Done())
		assert.IsType(w.GetDefer(), time.Now())
		assert.NoError(w.Do(ctx, nil))
	}
}
