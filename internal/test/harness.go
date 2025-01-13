package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/caryatid/cueball/state/blob"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/caryatid/cueball/state/log"
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
	var err error
	lm := make(map[string]cueball.Log)
	lm["fsys"], _  = log.NewFsys(ctx, "testdir")
	lm["pg"], _ = log.NewPG(ctx, dbconn)
	lm["mem"], _ = log.NewMem(ctx)
	pm := make(map[string]cueball.Pipe)
	pm["nats"], err = pipe.NewNats(ctx, natsconn)
	if err != nil {
		t.Error("failed: %v", err)
	}
	pm["fifo"], _  = pipe.NewFifo(ctx, "test.fifo", "testdir")
	pm["mem"], _  = pipe.NewMem(ctx)
	bm := make(map[string]cueball.Blob)
	bm["mem"], _ = blob.NewMem(ctx)
	m := make(map[string]cueball.State)
	for ln, l := range lm {
		for pn, p := range pm {
			for bn, b := range bm {
				m[ln + "." + pn + "." + bn], _ =
					state.NewState(ctx, p, l, b)
			}
		}
	}
	return m
}
