package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/worker"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"context"
	"time"
	"testing"
	"strings"
)

var (
	Logs = make(map[string]func (context.Context) cueball.Log)
	Pipes = make(map[string]func (context.Context) cueball.Pipe)
	Blobs = make(map[string]func (context.Context) cueball.Blob)
)


func TestStateComponents(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.Lc(ctx).Debug().Msg("HERE: " + "xxx")
	cueball.RegWorker(ctx, worker.NewTestWorker)
	for pn, pg := range Pipes {
		for ln, lg := range Logs {
			for bn, bg := range Blobs {
				tname := strings.Join([]string{pn,ln,bn}, "-")
				s, ctx := state.NewState(ctx, pg(ctx),
					lg(ctx),bg(ctx))
				t.Run(tname, func (t *testing.T) {
					err := TLog(ctx, s)
					assert.NoError(err)
					err = TPipe(ctx, s)
					assert.NoError(err)
				})
			}
		}
	}
}

func TLog(ctx context.Context, s cueball.State) error {
	store := s.Run(ctx, s.Store)
	checks := test.Wload(store)
	for !s.Check(ctx, checks) {
		for w := range s.Run(ctx, s.Scan) {
			if err := w.Do(ctx, s); err != nil {
				return err
			}
			store <- w
		}
	}
	err := s.Wait(ctx, time.Millisecond*250, checks) // NOTE: redundant
	for _, id := range checks {
		w, _ := s.Get(ctx, id)
		cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
	}
	return err
}

func TPipe(ctx context.Context, s cueball.State) error {
	ctx, _ = context.WithCancel(ctx)
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	checks := test.Wload(enq)
	for !s.Check(ctx, checks) {
		for w := range deq {
			if err := w.Do(ctx, s); err != nil {
				return err
			}
			if ! w.Done() { 
				enq <- w
			}
		}
		return nil
	}
	err := s.Wait(ctx, time.Millisecond*250, checks) // NOTE: redundant
	for _, id := range checks {
		w, _ := s.Get(ctx, id)
		cueball.Lc(ctx).Debug().Interface(" W ", w).Send()
	}
	return err
}

