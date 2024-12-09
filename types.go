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

// TODO Scanner && Valuer interfaces?
type Stage int

const (
	INIT Stage = iota
	RUNNING
	RETRY
	NEXT
	DONE
)

var StageStr = map[int]string{0: "INIT", 1: "RUNNING", 2: "RETRY", 3: "NEXT", 4: "DONE"}
var StageInt = map[string]int{"INIT": 0, "RUNNING": 1, "RETRY": 2, "NEXT": 3, "DONE": 4}

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
