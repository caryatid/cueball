//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/log"
//	"github.com/caryatid/cueball/state/blob"
	"os"
	"context"
)


func init () {
	dname, err := os.MkdirTemp("", "test")
	if err != nil {
		panic(err)
	}
	// TODO: shit
	// defer os.RemoveAll(dname)
	Logs["fsys"] = func (ctx context.Context) cueball.Log {
		l, err := log.NewFsys(ctx, dname)
		if err != nil {
			panic(err)
		}
		return l
	}
/*
	Blobs["fsys"] = func (ctx context.Context) cueball.Blob {
		l, err := blob.NewFsys(ctx, dname)
		if err != nil {
			panic(err)
		}
		return l
	}
*/
}
