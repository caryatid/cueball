//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/log"
	"github.com/caryatid/cueball/internal/test"
	"context"
)


func init () {
	Logs["pg"] = func (ctx context.Context) cueball.Log {
		l, err := log.NewPG(ctx, test.Dbconn)
		if err != nil {
			panic(err)
		}
		return l
	}
}
