package retry

import (
	"context"
	"errors"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/internal/test"
	"testing"
	"time"
)

func fail(ctx context.Context, s cueball.State) error {
	return errors.New("intentional fail")
}

func TestRetry(t *testing.T) {
	assert, ctx := test.TSetup(t)
	rc := NewCount(3, fail)[0]
	rb := NewBackoff(3, 4*time.Second, fail)[0]
	var dt time.Time
	for i := 0; i < 4; i++ {
		rc.Do(ctx, nil)
		rb.Do(ctx, nil)
		switch i {
		case 0, 1:
			assert.True(rc.Again(), "retry should be ok")
			dt = rb.Defer()
		case 2:
			assert.False(rc.Again(), "should be no more retry")
			assert.True(rb.Defer().After(dt), "timing is off")
		}
	}

}
