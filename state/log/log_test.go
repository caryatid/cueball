//go:build linux

package log

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
	"os"
	"testing"
)

func TestLog(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(worker.NewTestWorker)
	dname, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Errorf("dir fail %v", err)
	}
	defer os.RemoveAll(dname)
	tm := map[string]cueball.Log{
		"fsys": func() cueball.Log {
			p, err := NewFsys(ctx, dname)
			assert.NoError(err)
			return p
		}(),
		"pg": func() cueball.Log {
			p, err := NewPG(ctx, test.Dbconn)
			assert.NoError(err)
			return p
		}(),
		"mem": func() cueball.Log {
			p, err := NewMem(ctx)
			assert.NoError(err)
			return p
		}(),
	}
	for tname, l := range tm {
		var x []cueball.Worker
		for i := 0; i < 10; i++ {
			for _, wn := range cueball.Workers() {
				x = append(x, cueball.Gen(wn))
			}
		}
		t.Run(tname, func(t *testing.T) {
			s, _ := state.NewState(ctx, nil, l, nil)
			assert.NoError(test.logRun(ctx, s, x...))
		})
	}
}

func logRun(ctx context.Context, s cueball.State,
	ws ...cueball.Worker) error {
	store := s.Run(ctx, s.Store)
	var wids []uuid.UUID
	for _, w := range ws {
		wids = append(wids, w.ID())
		w.SetStatus(cueball.ENQUEUE)
		store <- w
	}
	g.Go(func () error {
		for w := range s.Run(ctx, s.Scan) {
			if err := w.Do(ctx, s); err != nil {
				return err
			}
			store <- w
		}
	})
	return s.Wait(ctx, time.Millisecond*250, wids)
}
