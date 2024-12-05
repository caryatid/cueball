package worker

import (
	"context"
	"cueball"
	"fmt"
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/rand"
)

type StageWorker struct {
	*Exec
	Word   string
	Number int
}

func (s *StageWorker) Name() string {
	return "stage-worker"
}

func (s *StageWorker) FuncInit() error {
	s.Load(s.Stage1, s.Stage2, s.Stage3)
	return nil
}

func (s *StageWorker) Printer() {
	log.Debug().Interface("stage", s).Send()
}

func (s *StageWorker) New() cueball.Worker {
	sw := &StageWorker{Exec: &Exec{}}
	sw.ID()
	return sw
}

func (s *StageWorker) Stage1(ctx context.Context) error {
	s.Number = rand.Int() % 10
	s.Printer()
	return nil
}

func (s *StageWorker) Stage2(ctx context.Context) error {
	s.Printer()
	if s.Number < 4 {
		s.Number = rand.Int() % 10
		return fmt.Errorf("an error")
	}
	return nil
}

func (s *StageWorker) Stage3(ctx context.Context) error {
	s.Printer()
	return nil
}
