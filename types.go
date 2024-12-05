package cueball

import (
	"encoding/json"
)

type EndError struct{}
type EnumError struct{}

func (e *EndError) Error() string {
	return "iteration complete"
}

func (e *EnumError) Error() string {
	return "invalid enum value"
}

type Stage int

const (
	RUNNING Stage = iota
	RETRY
	NEXT
	DONE
)

var StageStr = map[int]string{0: "RUNNING", 1: "RETRY", 2: "NEXT", 3: "DONE"}
var StageInt = map[string]int{"RUNNING": 0, "RETRY": 1, "NEXT": 2, "DONE": 3}

func (s *Stage) MarshalJSON() ([]byte, error) {
	ss := StageStr[int(*s)]
	return json.Marshal(ss)
}

func (s Stage) String() string {
	return StageStr[int(s)]
}

func (s *Stage) UnmarshalJSON(b []byte) error {
	i, ok := StageInt[string(b)]
	if !ok {
		return &EnumError{}
	}
	*s = Stage(i)
	return json.Unmarshal(b, s)
}
