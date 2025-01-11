package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

type harness struct {
	L *zerolog.Logger
	A *assert.Assertions
	C context.CancelFunc
	S map[string]cueball.State
}

func TSetup(t *testing.T) (h *harness, ctx context.Context) {
	l := zerolog.New(os.Stdout)
	h = new(harness)
	ctx, h.C = context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = l.WithContext(ctx)
	h.L = zerolog.Ctx(ctx)
	h.A = assert.New(t)
	h.S = make(map[string]cueball.State)
	return h, ctx
}

func AllThree(ctx context.Context) map[string]cueball.State {
	m := make(map[string]cueball.State)
	m["mem"], _ = state.NewMem(ctx)
	m["pg"], _ = state.NewPG(ctx,
		"postgresql://postgres:postgres@localhost:5432",
		"nats://localhost:4222")
	m["fifo"], _ = state.NewFifo(ctx, "fifo", ".test")
	return m
}
