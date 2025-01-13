package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/state/blob"
	"github.com/caryatid/cueball/state/log"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var dbconn = "postgresql://postgres:postgres@localhost:5432"
var natsconn = "nats://localhost:4222"

type harness struct {
	L *zerolog.Logger
	A *assert.Assertions
	C context.CancelFunc
}

func TSetup(t *testing.T) (h *harness, ctx context.Context) {
	l := zerolog.New(os.Stdout)
	h = new(harness)
	ctx, h.C = context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = l.WithContext(ctx)
	h.L = zerolog.Ctx(ctx)
	h.A = assert.New(t)
	return h, ctx
}

func AllThree(ctx context.Context, t *testing.T) map[string]cueball.State {
	m := make(map[string]cueball.State)
	pm := make(map[string]func() cueball.Pipe)
	lm := make(map[string]func() cueball.Log)
	pm["fifo"] = func() cueball.Pipe {
		p, _ := pipe.NewFifo(ctx, "test.fifo", "testdir")
		return p
	}
	pm["mem"] = func() cueball.Pipe {
		p, _ := pipe.NewMem(ctx)
		return p
	}
	pm["nats"] = func() cueball.Pipe {
		p, _ := pipe.NewNats(ctx, natsconn)
		return p
	}
	lm["fsys"] = func() cueball.Log {
		l, _ := log.NewFsys(ctx, "testdir")
		return l
	}
	lm["pg"] = func() cueball.Log {
		l, _ := log.NewPG(ctx, dbconn)
		return l
	}
	lm["mem"] = func() cueball.Log {
		l, _ := log.NewMem(ctx)
		return l
	}
	for _, ln := range []string{"fsys", "pg", "mem"} {
		for _, pn := range []string{"nats", "fifo", "mem"} {
			for _, bn := range []string{"mem"} {
				b, _ := blob.NewMem(ctx)
				m[ln+"."+pn+"."+bn], _ =
					state.NewState(ctx, pm[pn](), lm[ln](), b)
			}
		}
	}
	return m
}
