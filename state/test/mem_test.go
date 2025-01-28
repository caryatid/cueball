//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/log"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/caryatid/cueball/state/blob"
	"context"
)


func init () {
	Logs["mem"] = func (ctx context.Context) cueball.Log {
		l, err := log.NewMem(ctx)
		if err != nil {
			panic(err)
		}
		return l
	}
	Pipes["mem"] = func (ctx context.Context) cueball.Pipe {
		p, err := pipe.NewMem(ctx)
		if err != nil {
			panic(err)
		}
		return p
	}
	Blobs["mem"] = func (ctx context.Context) cueball.Blob {
		b, err := blob.NewMem(ctx)
		if err != nil {
			panic(err)
		}
		return b
	}
}
