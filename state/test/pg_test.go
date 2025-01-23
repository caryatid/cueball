//go:build linux

package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state/log"
)

func init() {
	Records["pg"] = func(ctx context.Context) cueball.Record {
		l, err := log.NewPG(ctx, test.Dbconn)
		if err != nil {
			panic(err)
		}
		return l
	}
}
