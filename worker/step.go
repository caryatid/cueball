package worker

import (
	"context"
	"github.com/caryatid/cueball"
)

type step struct {
	f         cueball.Method
	Error     cueball.Error
	NextI     *step `json:"next"`
	TriesI    int   `json:"tries"`
	CompleteI bool  `json:"complete"`
}

func (s *step) Load(ms ...cueball.Method) {
	for i, m := range ms {
		if i == 0 {
			s.f = m
			continue
		}
		if s.NextI == nil {
			s.NextI = &step{f: m}
		} else {
			s.NextI.f = m
		}
		s = s.NextI
	}
}

func (s *step) Current() cueball.Step {
	if !s.CompleteI || s.NextI == nil {
		return s
	}
	return s.NextI.Current()
}

func (s *step) Done() bool {
	ss := s
	for ss.NextI != nil {
		ss = ss.NextI
	}
	return ss.CompleteI
}

func (s *step) Next() cueball.Step {
	return s.NextI
}

func (s *step) Tries() int {
	return s.TriesI
}

func (s *step) Complete() bool {
	return s.CompleteI
}

func (s *step) Do(ctx context.Context) error {
	if s.CompleteI { // TODO how to handle per step done-ness
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
