package worker

import(
	"context"
	"github.com/caryatid/cueball"
)

type basicStep struct {
	stepStatus
	f cueball.Method
	next cueball.Step
}

type stepStatus struct {
	Error    cueball.Error
	TriesI int `json:"tries"`
	CompleteI bool `json:"complete"`
}

func (s *stepStatus)Tries() int {
	return s.TriesI
}

func (s *stepStatus)Complete() bool {
	return s.CompleteI
}

func (s *basicStep)Next() cueball.Step {
	return s.next
}

func (s *basicStep)Done() bool {
	ss := s.Current()
	return ss.Complete() && ss.Next() == nil
}

func (s *basicStep) Current() cueball.Step {
	if s.Next() == nil || s.Complete() == false {
		return s
	}
	return s.Next().Current()
}

func (s *basicStep) Do(ctx context.Context) error {
	if s.Complete() { // TODO how to handle per step done-ness
		return nil 
	}
	s.TriesI++
	err := s.f(ctx)
	if err != nil {
		s.Error = cueball.NewError(err)
	} else {
		s.CompleteI = true
	}
	return err
}

func (s *basicStep)Add(cs cueball.Step) cueball.Step {
	s.next = cs
	return s.next
}

func BasicStep(m cueball.Method) cueball.Step {
	return &basicStep{f: m}
}

type coreStep struct {
	stepStatus
	Name string
	next cueball.Step
}

func (s *coreStep)Next() cueball.Step {
	return s.next
}


func (s *coreStep)Done() bool {
	ss := s.Current()
	return ss.Complete() && ss.Next() == nil
}

func (s *coreStep) Current() cueball.Step {
	if s.Next() == nil || s.Complete() == false {
		return s
	}
	return s.Next().Current()
}

func (s *coreStep) Do(ctx context.Context) error {
	s.CompleteI = true
	return nil
}

func (s *coreStep)Add(cs cueball.Step) cueball.Step {
	s.next = cs
	return s.next
}

func CoreStep(name string) cueball.Step {
	return &coreStep{Name: name}
}

func Stepper(f func (cueball.Method) cueball.Step, ms ...cueball.Method) {
}
