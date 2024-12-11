package cueball

import (
	"encoding/json"
	"database/sql/driver"
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
	INIT Stage = iota
	RUNNING
	RETRY
	NEXT
	DONE
	FAIL
)

var stage2string = map[Stage]string{
	INIT: "INIT",
	RUNNING: "RUNNING",
	RETRY: "RETRY",
	NEXT: "NEXT",
	DONE: "DONE",
	FAIL: "FAIL"}
var string2stage = map[string]Stage{
	"INIT": INIT,
	"RUNNING": RUNNING,
	"RETRY": RETRY,
	"NEXT": NEXT,
	"DONE": DONE,
	"FAIL": FAIL}
func (s *Stage) MarshalJSON() ([]byte, error) {
	ss := stage2string[*s]
	return json.Marshal(ss)
}

func (s Stage) String() string {
	return stage2string[s]
}

func (s *Stage) UnmarshalJSON(b []byte) error {
	var ss string
	var ok bool
	if err := json.Unmarshal(b, &ss); err != nil {
		return err
	}
	*s, ok = string2stage[ss]
	if !ok {
		return &EnumError{}
	}
	return nil
}

func (s Stage) Value() (driver.Value, error) {
	return stage2string[s], nil
}

func (s *Stage) Scan(value interface{}) error {
	if value == nil {
		*s = INIT
		return nil
	}
	switch v := value.(type) {
	case string:	
		*s = string2stage[v]
	}
	return nil
}
