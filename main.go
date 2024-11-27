package main

import (
	_ "github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/rand"
	"cueball/execute"
	"fmt"
	"time"
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
	fmt.Printf("STAGE %d:%d (%d) %s\n", s.Count, s.Current, s.Number, s.ID())
}

func (s *StageWorker) New() execute.Worker {
	sw := StageWorker{Exec: &execute.Exec{}}
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

func main () {
	s, err := execute.NewFifoState("fifo", 1)
	if err != nil {
		log.Err(err).Send()
	}
	w := (&StageWorker{}).New()
	tick := time.NewTicker(5 * time.Millisecond)
	done := make(chan bool)
	go func () { 
		i:=0
		for  {
			select {
			case <- done:
			case <- tick.C:
				if i < 23 {
					s.Enqueue(w.New())
					i++
				}
			}
		}
	}()
	go func () {
		
		for {
			s.Dequeue(w)
		}
	
	}()
	execute.Run(s)
}
