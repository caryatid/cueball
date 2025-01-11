// Package cueball/state

package state

import (
	"context"
	"github.com/caryatid/cueball"
	"time"
)

func Start(ctx context.Context, s cueball.State) {
	t := time.NewTicker(time.Millisecond * 25)
	s.Group().Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				s.Group().Go(func() error {
					return s.LoadWork(ctx)
				})
			case w := <-s.Store():
				if err := s.Persist(ctx, w); err != nil {
					return err
				}
			case w := <-s.Work():
				s.Group().Go(func() error {
					w.Do(ctx) // error handled inside
					if !w.Done() {
						if cueball.DirectEnqueue {
							s.Enqueue(ctx, w)
						} else {
							w.SetStatus(cueball.ENQUEUE)
						}
					}
					s.Store() <- w
					return nil
				})
			}
		}
	})
	return
}

func Wait(ctx context.Context, s cueball.State, ws []cueball.Worker) error {
	tick := time.NewTicker(time.Millisecond * 250)
	for {
		select {
		case <-ctx.Done():
			s.Close()
			return nil
		case <-tick.C:
			gtg := true
			for _, w_ := range ws {
				w, err := s.Get(ctx, w_.ID())
				if err != nil || w == nil {
					continue
				}
				if !w.Done() {
					gtg = false
					break
				}
			}
			if gtg {
				c := make(chan error)
				go func() {
					defer close(c)
					c <- s.Group().Wait()
				}()
				select {
				case err, _ := <-c: // TODO handle ok
					return err
				case <-time.After(time.Millisecond * 750):
				}
				return nil
			}
		}
	}
}
