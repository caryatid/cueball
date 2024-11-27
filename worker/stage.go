package worker

import (
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"math/rand"
	"cueball"
	"cueball/execute"
	"fmt"
)


type StageWorker struct {
	*execute.Exec 
	Word string
	Number int
}


func (s *StageWorker)Name() string {
	return "stage-worker"
}

func (s *StageWorker)FuncInit() error {
	s.Load(s.Stage1, s.Stage2, s.Stage3)
	return nil
}

func (s *StageWorker) Printer() {
	log.Debug().Interface("stage", s).Send()
}

func (s *StageWorker) New() cueball.Worker {
	sw := StageWorker{Exec: &execute.Exec{}}
	sw.ID()
	sw.Group()
	return &sw
}

func (s *StageWorker) Stage1() error {
	s.Number = rand.Int() % 10
	s.Printer()
	return nil
}

func (s *StageWorker) Stage2() error {
	s.Printer()
	if s.Number < 4 {
		s.Number = rand.Int() % 10
		return fmt.Errorf("an error")
	}
	return nil
}

func (s *StageWorker) Stage3() error {
	s.Printer()
	return nil
}

