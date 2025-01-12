package retry

import (
	"context"
	"errors"
	"github.com/caryatid/cueball/internal/test"
	"testing"
	"time"
)

func fail(ctx context.Context) error {
	return errors.New("intentional fail")
}

func TestRetry(t *testing.T) {
	h, ctx := test.TSetup(t)
	rc := NewCount(3, fail)[0]
	rb := NewBackoff(3, 4*time.Second, fail)[0]
	var dt time.Time
	for i := 0; i < 4; i++ {
		rc.Do(ctx)
		rb.Do(ctx)
		switch i {
		case 0, 1:
			h.A.True(rc.Again(), "retry should be ok")
			dt = rb.Defer()
		case 2:
			h.A.False(rc.Again(), "should be no more retry")
			h.A.True(rb.Defer().After(dt), "timeing is off")
		}
	}

}
