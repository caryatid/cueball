//go:build linux

package state_test

import (
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state/pipe"
	"github.com/caryatid/cueball/internal/test"
	"context"
)


func init () {
	Pipes["nats"] = func (ctx context.Context) cueball.Pipe {
		p, err := pipe.NewNats(ctx, test.Natsconn)
		if err != nil {
			panic(err)
		}
		return p
	}
}
