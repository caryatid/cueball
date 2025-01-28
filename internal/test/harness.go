package test

import (
	"context"
	"github.com/caryatid/cueball"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

var (
Dbconn = "postgresql://postgres:postgres@localhost:5432"
Natsconn = "nats://localhost:4222"
)

func Wload (in chan<- cueball.Worker) []uuid.UUID {
	var checks []uuid.UUID
	for _, wname := range cueball.Workers() {
		w := cueball.GenWorker(wname)
		checks = append(checks, w.ID())
		in <- w
	}
	return checks
}

func TSetup(t *testing.T) (*assert.Assertions, context.Context) {
	ctx, _ := context.WithDeadline(context.Background(),
		time.Now().Add(time.Second*23))
	ctx = zerolog.New(os.Stdout).WithContext(ctx)
	return assert.New(t), ctx
}



