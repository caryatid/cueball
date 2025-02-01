//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/log"
	//	"github.com/caryatid/cueball/state/blob"
	"context"
	"os"
)

func init() {
	// TODO: shit
	// defer os.RemoveAll(dname)
	Records["fsys"] = func(ctx context.Context) cueball.Record {
		dname, err := os.MkdirTemp("", "test")
		if err != nil {
			panic(err)
		}
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
