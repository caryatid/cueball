//go:build linux

package pipe

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state"
	"os"
	"testing"
)

func TestPipe(t *testing.T) {
	h, ctx := test.TSetup(t)
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
				t.Errorf("fifo fail: %v", err)
				return nil
			}
			return p
		}(),
		"mem": func() cueball.Pipe {
			p, err := NewMem(ctx)
			if err != nil {
				t.Errorf("fifo fail: %v", err)
				return nil
			}
			return p
		}(),
	}
	for tname, p := range tm {
		t.Run(tname, func(t *testing.T) {
			s, _ := state.NewState(ctx, p, nil, nil)
			h.A.NoError(h.TestPipe(ctx, s))
		})
	}

}
