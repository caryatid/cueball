//go:build linux

package state_test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"github.com/caryatid/cueball/state/pipe"
)

func init() {
	Pipes["nats"] = func(ctx context.Context) cueball.Pipe {
		p, err := pipe.NewNats(ctx, test.Natsconn)
		if err != nil {
			panic(err)
		}
		return p
	}
}
