package log

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/google/uuid"
	"sync"
	"time"
)

var queue_size = 10 // TODO

type mem struct {
	ids sync.Map
}

func NewMem(ctx context.Context) (cueball.Record, error) {
	l := new(mem)
	return l, nil
}

func (l *mem) emulateSerialize(src, target cueball.Worker) error {
	b, _ := state.Marshal(src)
	return state.Unmarshal(string(b), target)
}

func (l *mem) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	w_, ok := l.ids.Load(uuid.String())
	if !ok {
		return nil, nil // TODO must error
	}
	w__, ok := w_.(cueball.Worker)
	if !ok {
		return nil, nil // TODO error
	}
	w := cueball.GenWorker(w__.Name())
	l.emulateSerialize(w__, w)
	return w, nil
}

func (l *mem) Store(ctx context.Context, ch chan cueball.Worker) error {
	for w := range ch {
		l.ids.Store(w.ID().String(), w)
	}
	return nil
}

func (l *mem) Scan(ctx context.Context, ch chan cueball.Worker) error {
	l.ids.Range(func(k, w_ any) bool {
		w, _ := w_.(cueball.Worker)
		if w.Status() == cueball.ENQUEUE &&
			w.GetDefer().Before(time.Now()) {
			w.SetStatus(cueball.INFLIGHT)
			ch <- w
		}
		return true
	})
	return nil
}

func (s *mem) Close() error {
	return nil
}
