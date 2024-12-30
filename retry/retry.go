package retry

import (
	"context"
	"github.com/caryatid/cueball"
	"math"
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

func (c *count) Do(ctx context.Context) error {
	c.Tries++
	return c.f(ctx)
}

func (c *count) Again() bool {
	return c.Tries <= c.Max
}

func (c *count) Defer() time.Time {
	return time.Now().Add(-time.Millisecond) // always be less
}

type backoff struct {
	*count
	InitialWindow time.Duration
}

func NewBackoff(max int, start_window time.Duration, fs ...cueball.Method) (rs []cueball.Retry) {
	for _, f := range fs {
		rs = append(rs, &backoff{count: &count{f: f, Max: max}, InitialWindow: start_window})
	}
	return
}

func (b *backoff) Defer() time.Time {
	return time.Now().Add(time.Duration(math.Pow(float64(b.InitialWindow),
		float64(b.Tries))))
}
