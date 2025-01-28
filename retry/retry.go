package retry

import (
	"context"
	"github.com/caryatid/cueball"
	"time"
)

type count struct {
	f     cueball.Method
	Tries int
	Max   int
}

func NewCount(max int, fs ...cueball.Method) (rs []cueball.Retry) {
	for _, f := range fs {
		rs = append(rs, &count{f: f, Max: max})
	}
	return
}

func (c *count) Do(ctx context.Context, s cueball.State) error {
	c.Tries++
	return c.f(ctx, s)
}

func (c *count) Again() bool {
	return c.Tries < c.Max
}

func (c *count) Name() string {
	return "count"
}

func (c *count) Defer() time.Time {
	return time.Now().Add(-time.Millisecond) // always be less
}

type backoff struct {
	*count
	Window time.Duration
}

func NewBackoff(max int, start_window time.Duration, fs ...cueball.Method) (rs []cueball.Retry) {
	for _, f := range fs {
		rs = append(rs, &backoff{count: &count{f: f, Max: max}, Window: start_window})
	}
	return
}

func (b *backoff) Do(ctx context.Context, s cueball.State) error {
	b.Window = b.Window + (b.Window * time.Duration(b.Tries))
	b.Tries++
	return b.f(ctx, s)
}

func (b *backoff) Defer() time.Time {
	t := time.Now().Add(b.Window)
	return t
}

func (b *backoff) Name() string {
	return "backoff"
}
