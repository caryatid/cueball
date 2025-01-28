//go:build linux

package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state/log"
)

func init() {
	Logs["pg"] = func(ctx context.Context) cueball.Log {
		l, err := log.NewPG(ctx, test.Dbconn)
		if err != nil {
			panic(err)
		}
		return l
	}
}
