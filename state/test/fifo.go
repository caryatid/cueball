//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/pipe"
	"context"
	"os"
)


func init () {
	dname, err := os.MkdirTemp("", "test")
	if err != nil {
		panic(err)
	}
	Pipes["fifo"] = func (ctx context.Context) cueball.Pipe {
		p, err := pipe.NewFifo(ctx, "fifo-test", dname)
		if err != nil {
			panic(err)
		}
		return p
	}
}
