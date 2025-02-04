package worker

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"time"
)

type defaultExecutor struct {
	Steps   []*step
	Id      uuid.UUID
	StatusI cueball.Status `json:"status"`
	// TODO version
}

type step struct {
	Attempt  cueball.Retry
	Success  bool
	Complete bool
	Error    *cueball.Error
}

func NewExecutor(rs ...cueball.Retry) cueball.Executor {
	e := new(defaultExecutor)
	e.Id, _ = uuid.NewRandom() // TODO error handling
	for _, r := range rs {
		e.Steps = append(e.Steps, &step{Attempt: r})
	}
	return e
}

func (e *defaultExecutor) ID() uuid.UUID {
	return e.Id
}

func (e *defaultExecutor) Status() cueball.Status {
	return e.StatusI
}

func (e *defaultExecutor) SetStatus(s cueball.Status) {
	e.StatusI = s
}

func (e *defaultExecutor) Done() bool {
	return e.StatusI == cueball.FAIL || e.Steps[len(e.Steps)-1].Complete
}

func (e *defaultExecutor) GetDefer() time.Time {
	return e.current().Attempt.Defer()
}

func (e *defaultExecutor) Do(ctx context.Context, s cueball.State) error {
	st := e.current()
	if st.Complete {
		return st.Error
	}
	st.Error = nil // NOTE clobbers previous error, for this step. Could be list?
	err := st.Attempt.Do(ctx, s)
	if err != nil {
		st.Error = cueball.NewError(err)
		if !st.Attempt.Again() {
			st.Success = false // explicit but should not be necessary
			e.StatusI = cueball.FAIL
			return err
		}
	}
	st.Success = true
	st.Complete = true
	if e.Done() {
		e.StatusI = cueball.DONE
	} else if cueball.DirectEnqueue {
		e.StatusI = cueball.INFLIGHT
	} else {
		e.StatusI = cueball.ENQUEUE
	}
	return err // returns error even if retry is gtg
}

func (e *defaultExecutor) current() *step {
	var rets *step
	for _, s := range e.Steps {
		if !s.Complete {
			return s
		}
		rets = s
	}
	return rets
}
