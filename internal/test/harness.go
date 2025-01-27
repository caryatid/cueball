package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"os"
	"testing"
	"time"
)

type pf func () cueball.Pipe
type lf func () cueball.Log
type bf func () cueball.Blob
var (
Dbconn = "postgresql://postgres:postgres@localhost:5432"
Natsconn = "nats://localhost:4222"
Pipes = map[string] pf {
		"fifo": pf {
			p, err := NewFifo(ctx, "test.fifo", dname)
			if err != nil {
				panic(err)
			}
			return p
		},
		"nats": pf {
			p, err := NewNats(ctx, test.Natsconn)
			if err != nil {
				panic(err)
			}
			return p
		},
		"mem": pf {
			p, err := NewMem(ctx)
			if err != nil {
				panic(err)
			}
			return p
		},
	}

Logs = map[string]lf {
		"fsys": lf {
			l, err := NewFsys(ctx, dname)
			if err != nil {
				panic(err)
			}
			return l
		},
		"pg": lf {
			l, err := NewPG(ctx, test.Dbconn)
			if err != nil {
				panic(err)
			}
			return l
		},
		"mem": lf {
			l, err := NewMem(ctx)
			if err != nil {
				panic(err)
			}
			return l
		},
	}
)

func TSetup(t *testing.T) (*assert.Assertions, context.Context) {
	ctx, _ := context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = zerolog.New(os.Stdout).WithContext(ctx)
	return assert.New(t), ctx
}



