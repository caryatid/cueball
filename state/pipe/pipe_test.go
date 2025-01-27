//go:build linux

package pipe

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
	"fmt"
	"os"
	"testing"
)

func TestPipe(t *testing.T) {
	assert, ctx := test.TSetup(t)
	cueball.RegGen(worker.NewTestWorker)
	dname, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Errorf("dir fail %v", err)
	}
	defer os.RemoveAll(dname)
	tm := map[string]cueball.Pipe{
	}
	for tname, p := range tm {
		var x []cueball.Worker
		for i := 0; i < 10; i++ {
			for _, wn := range cueball.Workers() {
				x = append(x, cueball.Gen(wn))
			}
		}
		t.Run(tname, func(t *testing.T) {
			s, ctx := state.NewState(ctx, p, nil, nil)
			cueball.Lc(ctx).Debug().
				Str("pipe", fmt.Sprintf("pipe: %s, %v", tname, p))
			assert.NoError(pipeRun(ctx, s, x...))
		})
	}
}


func pipeRun(ctx context.Context, s cueball.State, ws ...cueball.Worker) error {
	ctx, c := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	enq := s.Run(ctx, s.Enqueue)
	deq := s.Run(ctx, s.Dequeue)
	g.Go(func() error {
		defer func() {
			time.Sleep(time.Millisecond * 100)
			s.Close()
			c()
		}()
		for _, w := range ws {
			enq <- w
		}
		close(enq)
		return nil
	})
	g.Go(func() error {
		for w := range deq {
			if err := w.Do(ctx, s); err != nil {
				return err
			}
			if ! w.Done() { 
				enq <- w
			}
		}
		return nil
	})
	return g.Wait()
}

