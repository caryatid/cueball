// Package cueball/state

package state

import (
	"context"
	"github.com/caryatid/cueball"
	"golang.org/x/sync/errgroup"
	"time"
)

func Start(ctx_ context.Context, s cueball.State) (g *errgroup.Group,
	ctx context.Context) {
	g, ctx = errgroup.WithContext(ctx_)
	t := time.NewTicker(time.Millisecond * 25)
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				g.Go(func() error {
					return s.LoadWork(ctx)
				})
			case w := <-s.Store():
				if err := s.Persist(ctx, w); err != nil {
					return err
				}
			case w := <-s.Work():
				g.Go(func() error {
					w.Do(ctx) // error handled inside
					if !w.Done() {
						if cueball.DirectEnqueue {
							s.Enqueue(ctx, w)
						} else {
							w.SetStatus(cueball.ENQUEUE)
						}
					}
					// cueball.Lc(ctx).Debug().Interface("W", w).Send()
					s.Store() <- w
					return nil
				})
			}
		}
	})
	return
}
