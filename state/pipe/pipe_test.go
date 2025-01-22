//go:build linux

package pipe

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/worker"
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
		"fifo": func() cueball.Pipe {
			p, err := NewFifo(ctx, "test.fifo", dname)
			if err != nil {
				t.Errorf("fifo fail: %v", err)
				return nil
			}
			return p
		}(),
		"nats": func() cueball.Pipe {
			p, err := NewNats(ctx, test.Natsconn)
			if err != nil {
				t.Errorf("nats fail: %v", err)
				return nil
			}
			return p
		}(),
		"mem": func() cueball.Pipe {
			p, err := NewMem(ctx)
			if err != nil {
				t.Errorf("mem fail: %v", err)
				return nil
			}
			return p
		}(),
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
			assert.NoError(test.Pipe(ctx, s, x...))
		})
	}
}
