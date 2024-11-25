package main

import (
	"cueball/execute"
	"fmt"
)


type StageWorker struct {
	*execute.Exec 
	word string
	number int
}

func (s *StageWorker)Name() string {
	return "stage-worker"
}

func (s *StageWorker)FuncInit() error {
	s.Load(s.Stage1, s.Stage2, s.Stage3)
	return nil
}

func (s *StageWorker) New() execute.Worker {
	return &StageWorker{Exec: new(execute.Exec)}
}

func (s *StageWorker) Stage1() error {
	fmt.Println("STAGE ONE")
	return nil
}

func (s *StageWorker) Stage2() error {
	fmt.Println("STAGE TWO")
	return nil
}

func (s *StageWorker) Stage3() error {
	fmt.Println("STAGE THREE")
	return nil
}


func main () {
	sw := &StageWorker{Exec: new(execute.Exec), word: "hello", number: 123}
	s, err := execute.NewFifoState("fifo", 1)
	if err != nil {
		fmt.Errorf("WOAH: %w\n", err)
	}
	s.Enqueue(sw)
	execute.Run(s, sw)
}
